package stream_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/temic/go-music/internal/stream"
)

func TestFileWithInfoUsesCachedSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.mp3")
	content := []byte("0123456789abcdef")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("Range", "bytes=0-4")
	rec := httptest.NewRecorder()

	err := stream.FileWithInfo(rec, req, stream.FileInfo{
		Path:    path,
		Size:    int64(len(content)),
		ModTime: time.Unix(1_700_000_000, 0).UTC(),
		Format:  "mp3",
	})
	if err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusPartialContent)
	}
	if got := rec.Body.String(); got != "01234" {
		t.Fatalf("body = %q, want %q", got, "01234")
	}
	if rec.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("content-type = %q", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Header().Get("Content-Range"), "bytes 0-4/16") {
		t.Fatalf("content-range = %q", rec.Header().Get("Content-Range"))
	}
}
