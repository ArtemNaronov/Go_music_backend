package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFlacDuration(t *testing.T) {
	path := findSampleFlac(t)
	if path == "" {
		t.Skip("no flac file in D:\\Music")
	}

	duration, err := readDuration(path, "flac")
	if err != nil {
		t.Fatal(err)
	}

	if duration < 30 || duration > 3600 {
		t.Fatalf("duration = %f, want between 30s and 1h", duration)
	}
}

func findSampleFlac(t *testing.T) string {
	t.Helper()
	root := `D:\Music`
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".flac") {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
