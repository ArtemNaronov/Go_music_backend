package scanner

import (
	"context"
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
)

type recordingIndexer struct {
	scans    []string
	removals []string
}

func (r *recordingIndexer) ScanFile(_ context.Context, path string) error {
	r.scans = append(r.scans, path)
	return nil
}

func (r *recordingIndexer) RemovePath(_ context.Context, path string) error {
	r.removals = append(r.removals, path)
	return nil
}

func TestWatcherDebounce(t *testing.T) {
	indexer := &recordingIndexer{}
	watcher := NewWatcher(`D:\Music`, indexer, zerolog.Nop())

	path := `D:\Music\Artist\Song.mp3`
	watcher.schedule(path, fsnotify.Remove)
	watcher.schedule(path, fsnotify.Remove)
	watcher.flush(context.Background())

	if len(indexer.removals) != 1 {
		t.Fatalf("removals = %d, want 1", len(indexer.removals))
	}
}

func TestMergeOpsPrefersRemove(t *testing.T) {
	op := mergeOps(fsnotify.Write, fsnotify.Remove)
	if !op.Has(fsnotify.Remove) {
		t.Fatal("expected remove op to be preserved")
	}
}
