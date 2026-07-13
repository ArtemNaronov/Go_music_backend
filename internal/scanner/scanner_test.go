package scanner

import (
	"bytes"
	"io"
	"testing"

	"github.com/temic/go-music/pkg/musicpath"
)

func TestTitleFromFilename(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"01 - Foreword.mp3", "Foreword"},
		{"02.mp3", "02"},
		{"Bohemian Rhapsody.flac", "Bohemian Rhapsody"},
	}

	for _, tc := range tests {
		if got := titleFromFilename(tc.in); got != tc.want {
			t.Fatalf("titleFromFilename(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFoldersFromPath(t *testing.T) {
	root := `D:\Music`
	path := `D:\Music\Linkin Park\Meteora\01 - Foreword.mp3`

	artist, album := musicpath.FolderArtistAlbum(root, path)
	if artist != "Linkin Park" || album != "Meteora" {
		t.Fatalf("folders = (%q, %q), want (Linkin Park, Meteora)", artist, album)
	}
}

func TestWavDuration(t *testing.T) {
	data := []byte{
		'R', 'I', 'F', 'F', 36, 0, 0, 0,
		'W', 'A', 'V', 'E',
		'f', 'm', 't', ' ', 16, 0, 0, 0,
		1, 0, 1, 0,
		0x44, 0xAC, 0, 0,
		0x88, 0x58, 1, 0,
		2, 0, 16, 0,
		'd', 'a', 't', 'a', 4, 0, 0, 0,
		0, 0, 0, 0,
	}

	reader := bytes.NewReader(data)
	duration, err := wavDuration(reader)
	if err != nil {
		t.Fatal(err)
	}

	if duration <= 0 {
		t.Fatalf("duration = %f, want > 0", duration)
	}

	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
}
