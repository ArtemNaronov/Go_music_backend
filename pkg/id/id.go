package id

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

// Track derives a stable identifier from an absolute file path.
func Track(path string) string {
	path = strings.ToLower(filepath.Clean(path))
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:])
}

// Album derives a stable identifier from artist and album names.
func Album(artist, album string) string {
	key := strings.ToLower(strings.TrimSpace(artist)) + "\x00" + strings.ToLower(strings.TrimSpace(album))
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:16])
}

// AlbumDir derives a stable identifier from a relative album directory path.
func AlbumDir(relDir string) string {
	relDir = strings.ToLower(filepath.Clean(relDir))
	sum := sha256.Sum256([]byte(relDir))
	return hex.EncodeToString(sum[:16])
}

// Artist derives a stable identifier from an artist name.
func Artist(name string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name))))
	return hex.EncodeToString(sum[:16])
}
