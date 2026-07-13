package stream

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var contentTypesByExt = map[string]string{
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	".m4a":  "audio/mp4",
	".wav":  "audio/wav",
}

var contentTypesByFormat = map[string]string{
	"mp3":  "audio/mpeg",
	"flac": "audio/flac",
	"m4a":  "audio/mp4",
	"wav":  "audio/wav",
}

// FileInfo describes a track file for streaming without an extra Stat syscall.
type FileInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
	Format  string
}

// File streams an audio file with HTTP Range Request support.
func File(w http.ResponseWriter, r *http.Request, path string) error {
	return FileWithInfo(w, r, FileInfo{Path: path})
}

// FileWithInfo streams using cached metadata from the library index.
func FileWithInfo(w http.ResponseWriter, r *http.Request, info FileInfo) error {
	path := info.Path
	if path == "" {
		return os.ErrNotExist
	}

	raw, err := os.Open(path)
	if err != nil {
		return err
	}
	defer raw.Close()

	var content io.ReadSeeker = raw
	if info.Size > 0 {
		content = &cachedStatFile{
			File:    raw,
			size:    info.Size,
			modTime: info.ModTime,
			name:    filepath.Base(path),
		}
	}

	contentType := contentTypeFor(info.Format, path)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "public, max-age=86400")

	modTime := info.ModTime
	if modTime.IsZero() {
		if stat, err := raw.Stat(); err == nil {
			modTime = stat.ModTime()
		}
	}

	http.ServeContent(w, r, filepath.Base(path), modTime, content)
	return nil
}

func contentTypeFor(format, path string) string {
	if mime, ok := contentTypesByFormat[strings.ToLower(format)]; ok {
		return mime
	}

	ext := strings.ToLower(filepath.Ext(path))
	if mime, ok := contentTypesByExt[ext]; ok {
		return mime
	}

	return "application/octet-stream"
}

type cachedStatFile struct {
	*os.File
	size    int64
	modTime time.Time
	name    string
}

func (f *cachedStatFile) Stat() (os.FileInfo, error) {
	if f.size > 0 {
		return &cachedFileInfo{
			name:    f.name,
			size:    f.size,
			modTime: f.modTime,
		}, nil
	}
	return f.File.Stat()
}

type cachedFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (i *cachedFileInfo) Name() string       { return i.name }
func (i *cachedFileInfo) Size() int64        { return i.size }
func (i *cachedFileInfo) Mode() os.FileMode  { return 0o644 }
func (i *cachedFileInfo) ModTime() time.Time { return i.modTime }
func (i *cachedFileInfo) IsDir() bool        { return false }
func (i *cachedFileInfo) Sys() any           { return nil }
