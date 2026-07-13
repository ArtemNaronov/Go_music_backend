package scanner

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/tcolgate/mp3"
)

func readDuration(path, format string) (float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	return readDurationFrom(file, format)
}

func readDurationFrom(r io.ReadSeeker, format string) (float64, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	switch format {
	case "mp3":
		return mp3Duration(r)
	case "flac":
		return flacDuration(r)
	case "m4a":
		return m4aDuration(r)
	case "wav":
		return wavDuration(r)
	default:
		return 0, fmt.Errorf("unsupported format: %s", format)
	}
}

func mp3Duration(r io.ReadSeeker) (float64, error) {
	decoder := mp3.NewDecoder(r)

	var duration float64
	for {
		var frame mp3.Frame
		var skipped int
		// tcolgate/mp3 panics if skipped is nil — always pass a pointer.
		if err := decoder.Decode(&frame, &skipped); err == io.EOF {
			break
		} else if err != nil {
			return 0, err
		}
		duration += frame.Duration().Seconds()
	}

	return duration, nil
}

func flacDuration(r io.ReadSeeker) (float64, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	var marker [4]byte
	if _, err := io.ReadFull(r, marker[:]); err != nil {
		return 0, err
	}
	if string(marker[:]) != "fLaC" {
		return 0, fmt.Errorf("invalid flac header")
	}

	for {
		var blockHeader [4]byte
		if _, err := io.ReadFull(r, blockHeader[:]); err != nil {
			return 0, err
		}

		isLast := blockHeader[0]&0x80 != 0
		blockType := blockHeader[0] & 0x7F
		length := uint32(blockHeader[1])<<16 | uint32(blockHeader[2])<<8 | uint32(blockHeader[3])

		if blockType == 0 {
			if length < 34 {
				return 0, fmt.Errorf("invalid streaminfo block")
			}

			info := make([]byte, 34)
			if _, err := io.ReadFull(r, info); err != nil {
				return 0, err
			}
			if length > 34 {
				if _, err := r.Seek(int64(length-34), io.SeekCurrent); err != nil {
					return 0, err
				}
			}

			sampleRate := uint32(info[10])<<12 | uint32(info[11])<<4 | uint32(info[12])>>4
			totalSamples := uint64(info[13]&0x0F)<<32 |
				uint64(info[14])<<24 |
				uint64(info[15])<<16 |
				uint64(info[16])<<8 |
				uint64(info[17])

			if sampleRate == 0 {
				return 0, fmt.Errorf("invalid flac sample rate")
			}

			return float64(totalSamples) / float64(sampleRate), nil
		}

		if _, err := r.Seek(int64(length), io.SeekCurrent); err != nil {
			return 0, err
		}

		if isLast {
			break
		}
	}

	return 0, fmt.Errorf("streaminfo block not found")
}

func wavDuration(r io.ReadSeeker) (float64, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	var riffHeader [12]byte
	if _, err := io.ReadFull(r, riffHeader[:]); err != nil {
		return 0, err
	}

	if string(riffHeader[0:4]) != "RIFF" || string(riffHeader[8:12]) != "WAVE" {
		return 0, fmt.Errorf("invalid wav header")
	}

	var byteRate uint32
	var dataSize uint32

	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(r, chunkHeader[:]); err != nil {
			return 0, err
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			fmtChunk := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, fmtChunk); err != nil {
				return 0, err
			}
			if len(fmtChunk) >= 16 {
				byteRate = binary.LittleEndian.Uint32(fmtChunk[8:12])
			}
		case "data":
			dataSize = chunkSize
			if byteRate == 0 {
				return 0, fmt.Errorf("wav fmt chunk missing")
			}
			return float64(dataSize) / float64(byteRate), nil
		default:
			if _, err := r.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return 0, err
			}
		}

		if chunkSize%2 == 1 {
			if _, err := r.Seek(1, io.SeekCurrent); err != nil {
				return 0, err
			}
		}
	}
}

func m4aDuration(r io.ReadSeeker) (float64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}

	timescale, duration, ok := parseMVHD(data)
	if !ok {
		return 0, fmt.Errorf("mvhd atom not found")
	}
	if timescale == 0 {
		return 0, fmt.Errorf("invalid m4a timescale")
	}

	return float64(duration) / float64(timescale), nil
}

func parseMVHD(data []byte) (timescale uint32, duration uint64, ok bool) {
	offset := 0
	for offset+8 <= len(data) {
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		atomType := string(data[offset+4 : offset+8])
		if size < 8 {
			return 0, 0, false
		}

		end := offset + size
		if end > len(data) {
			return 0, 0, false
		}

		payload := data[offset+8 : end]
		switch atomType {
		case "moov", "trak", "mdia":
			if childTimescale, childDuration, found := parseMVHD(payload); found {
				return childTimescale, childDuration, true
			}
		case "mvhd":
			return parseMVHDPayload(payload)
		}

		offset = end
	}

	return 0, 0, false
}

func parseMVHDPayload(payload []byte) (uint32, uint64, bool) {
	if len(payload) < 20 {
		return 0, 0, false
	}

	version := payload[0]
	switch version {
	case 0:
		if len(payload) < 24 {
			return 0, 0, false
		}
		timescale := binary.BigEndian.Uint32(payload[12:16])
		duration := uint64(binary.BigEndian.Uint32(payload[16:20]))
		return timescale, duration, true
	case 1:
		if len(payload) < 32 {
			return 0, 0, false
		}
		timescale := binary.BigEndian.Uint32(payload[20:24])
		duration := binary.BigEndian.Uint64(payload[24:32])
		return timescale, duration, true
	default:
		return 0, 0, false
	}
}
