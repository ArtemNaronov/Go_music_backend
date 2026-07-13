package cover

import (
	"os"
	"path/filepath"
	"strings"
)

var folderCoverNames = []string{"cover.jpg", "folder.jpg", "front.jpg"}

// FromFolder loads the first available folder artwork near a track.
func FromFolder(trackPath string) ([]byte, string, bool) {
	dir := filepath.Dir(trackPath)

	for _, name := range folderCoverNames {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		return data, mimeFromExt(filepath.Ext(name)), true
	}

	return nil, "", false
}

func mimeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
