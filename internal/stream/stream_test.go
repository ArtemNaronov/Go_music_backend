package stream

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSupportsRangeRequests(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.mp3")
	content := []byte("0123456789abcdef")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("Range", "bytes=0-4")
	rec := httptest.NewRecorder()

	if err := File(rec, req, path); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusPartialContent)
	}

	if got := rec.Body.String(); got != "01234" {
		t.Fatalf("body = %q, want %q", got, "01234")
	}

	if !strings.Contains(rec.Header().Get("Content-Range"), "bytes 0-4/") {
		t.Fatalf("unexpected content-range: %s", rec.Header().Get("Content-Range"))
	}
}

func TestFileSupportsHead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.flac")
	if err := os.WriteFile(path, []byte("flac-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodHead, "/stream", nil)
	rec := httptest.NewRecorder()

	if err := File(rec, req, path); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body length = %d, want 0", rec.Body.Len())
	}
	if rec.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("missing Accept-Ranges header")
	}
}
