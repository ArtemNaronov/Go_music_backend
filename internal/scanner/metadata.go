package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
	"github.com/temic/go-music/pkg/musicpath"
)

type parsedTrack struct {
	track models.Track
	cover []byte
	mime  string
}

func parseFile(musicRoot, path, format string) (parsedTrack, error) {
	info, err := os.Stat(path)
	if err != nil {
		return parsedTrack{}, err
	}

	filename := filepath.Base(path)
	folderArtist, folderAlbum := musicpath.FolderArtistAlbum(musicRoot, path)

	title := titleFromFilename(filename)
	artist := folderArtist
	album := folderAlbum
	year := 0
	genre := ""
	trackNumber := 0
	discNumber := 0
	hasCover := false

	var coverData []byte
	var coverMIME string

	file, err := os.Open(path)
	if err != nil {
		return parsedTrack{}, err
	}
	defer file.Close()

	if metadata, err := tag.ReadFrom(file); err == nil {
		title = firstNonEmpty(metadata.Title(), title)
		artist = firstNonEmpty(metadata.Artist(), metadata.AlbumArtist(), artist)
		album = firstNonEmpty(metadata.Album(), album)
		year = metadata.Year()
		genre = metadata.Genre()

		trackNumber, _ = metadata.Track()
		discNumber, _ = metadata.Disc()

		if picture := metadata.Picture(); picture != nil && len(picture.Data) > 0 {
			hasCover = true
			coverData = append([]byte(nil), picture.Data...)
			coverMIME = picture.MIMEType
			if coverMIME == "" {
				coverMIME = mimeFromPictureExt(picture.Ext)
			}
		}
	}

	if _, err := file.Seek(0, 0); err != nil {
		return parsedTrack{}, err
	}

	duration, err := readDurationFrom(file, format)
	if err != nil {
		duration = 0
	}

	if title == "" {
		title = filename
	}
	if artist == "" {
		artist = "Unknown Artist"
	}
	if album == "" {
		album = "Unknown Album"
	}

	track := models.Track{
		ID:          id.Track(path),
		Path:        path,
		Title:       title,
		Artist:      artist,
		Album:       album,
		Year:        year,
		Genre:       genre,
		Duration:    duration,
		TrackNumber: trackNumber,
		DiscNumber:  discNumber,
		HasCover:    hasCover,
		Size:        info.Size(),
		Format:      format,
		ModifiedAt:  info.ModTime().UTC(),
	}

	return parsedTrack{
		track: track,
		cover: coverData,
		mime:  coverMIME,
	}, nil
}

func mimeFromPictureExt(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func hasFolderCover(path string) bool {
	dir := filepath.Dir(path)
	for _, name := range []string{"cover.jpg", "folder.jpg", "front.jpg"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func enrichFolderCover(track models.Track) models.Track {
	if !track.HasCover && hasFolderCover(track.Path) {
		track.HasCover = true
	}
	return track
}
