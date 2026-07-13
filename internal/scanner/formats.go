package scanner

import (
	"path/filepath"
	"strings"
)

var supportedExtensions = map[string]string{
	".mp3":  "mp3",
	".flac": "flac",
	".m4a":  "m4a",
	".wav":  "wav",
}

func isSupported(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	format, ok := supportedExtensions[ext]
	return format, ok
}
