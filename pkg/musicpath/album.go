package musicpath

import (
	"path/filepath"
	"regexp"
	"strings"
)

var discFolderPattern = regexp.MustCompile(`(?i)^(cd|disc|disk)[\s._-]*\d+$`)

// RelAlbumDir returns a normalized album directory path relative to the music root.
// Disc subfolders such as CD1 or Disc 2 are collapsed into the parent album folder.
func RelAlbumDir(root, trackPath string) string {
	parts := relDirParts(root, trackPath)
	if len(parts) == 0 {
		return "_root"
	}

	if len(parts) > 1 && isDiscFolder(parts[len(parts)-1]) {
		parts = parts[:len(parts)-1]
	}

	return strings.ToLower(filepath.Join(parts...))
}

// FolderArtistAlbum derives artist and album names from the folder structure.
func FolderArtistAlbum(root, trackPath string) (artist, album string) {
	parts := relDirParts(root, trackPath)
	if len(parts) == 0 {
		return "Unknown Artist", "Unknown Album"
	}

	if len(parts) > 1 && isDiscFolder(parts[len(parts)-1]) {
		parts = parts[:len(parts)-1]
	}

	switch len(parts) {
	case 1:
		return parts[0], "Unknown Album"
	default:
		return parts[0], parts[len(parts)-1]
	}
}

func relDirParts(root, trackPath string) []string {
	rel, err := filepath.Rel(root, filepath.Dir(trackPath))
	if err != nil || rel == "." {
		return nil
	}

	return strings.Split(filepath.ToSlash(rel), "/")
}

func isDiscFolder(name string) bool {
	return discFolderPattern.MatchString(strings.TrimSpace(name))
}
