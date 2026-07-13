package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
)

const defaultDebounce = 500 * time.Millisecond

// FileIndexer handles incremental library updates for individual paths.
type FileIndexer interface {
	ScanFile(ctx context.Context, path string) error
	RemovePath(ctx context.Context, path string) error
}

// Watcher monitors the music directory and updates the library on changes.
type Watcher struct {
	musicPath string
	indexer   FileIndexer
	logger    zerolog.Logger
	debounce  time.Duration

	mu      sync.Mutex
	pending map[string]fsnotify.Op
	timer   *time.Timer
}

// NewWatcher creates a filesystem watcher with debounced event handling.
func NewWatcher(
	musicPath string,
	indexer FileIndexer,
	logger zerolog.Logger,
) *Watcher {
	return &Watcher{
		musicPath: filepath.Clean(musicPath),
		indexer:   indexer,
		logger:    logger.With().Str("component", "watcher").Logger(),
		debounce:  defaultDebounce,
		pending:   make(map[string]fsnotify.Op),
	}
}

// Run watches the music directory until the context is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := w.addRecursive(watcher, w.musicPath); err != nil {
		return err
	}

	w.logger.Info().Str("path", w.musicPath).Msg("filesystem watcher started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("filesystem watcher stopped")
			w.stopTimer()
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(watcher, event)

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			w.logger.Error().Err(err).Msg("filesystem watcher error")
		}
	}
}

func (w *Watcher) handleEvent(watcher *fsnotify.Watcher, event fsnotify.Event) {
	if !w.isRelevantPath(event.Name) {
		return
	}

	if event.Has(fsnotify.Create) {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			if err := w.addRecursive(watcher, event.Name); err != nil {
				w.logger.Warn().Err(err).Str("path", event.Name).Msg("failed to watch directory")
			}
		}
	}

	w.schedule(event.Name, event.Op)
}

func (w *Watcher) schedule(path string, op fsnotify.Op) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if existing, ok := w.pending[path]; ok {
		w.pending[path] = mergeOps(existing, op)
	} else {
		w.pending[path] = op
	}

	if w.timer != nil {
		w.timer.Stop()
	}

	w.timer = time.AfterFunc(w.debounce, func() {
		w.flush(context.Background())
	})
}

func (w *Watcher) flush(ctx context.Context) {
	w.mu.Lock()
	batch := w.pending
	w.pending = make(map[string]fsnotify.Op)
	w.timer = nil
	w.mu.Unlock()

	for path, op := range batch {
		w.processPath(ctx, path, op)
	}
}

func (w *Watcher) processPath(ctx context.Context, path string, op fsnotify.Op) {
	path = filepath.Clean(path)

	if op.Has(fsnotify.Remove) {
		if err := w.indexer.RemovePath(ctx, path); err != nil {
			w.logger.Warn().Err(err).Str("path", path).Msg("failed to remove track")
			return
		}
		w.logger.Info().Str("path", path).Msg("track removed")
		return
	}

	if _, err := os.Stat(path); err != nil {
		return
	}

	if !isAudioFile(path) {
		return
	}

	if err := w.indexer.ScanFile(ctx, path); err != nil {
		w.logger.Warn().Err(err).Str("path", path).Msg("failed to index track")
		return
	}

	w.logger.Info().Str("path", path).Str("op", op.String()).Msg("track updated")
}

func (w *Watcher) addRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			w.logger.Warn().Err(err).Str("path", path).Msg("failed to access path while watching")
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if err := watcher.Add(path); err != nil {
			return err
		}
		return nil
	})
}

func (w *Watcher) isRelevantPath(path string) bool {
	path = filepath.Clean(path)
	if !isWithinRoot(w.musicPath, path) {
		return false
	}

	if isAudioFile(path) {
		return true
	}

	info, err := os.Stat(path)
	if err != nil {
		return isAudioFile(path)
	}

	return info.IsDir()
}

func isWithinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isAudioFile(path string) bool {
	_, ok := isSupported(path)
	return ok
}

func mergeOps(existing, incoming fsnotify.Op) fsnotify.Op {
	if incoming.Has(fsnotify.Remove) || incoming.Has(fsnotify.Rename) {
		return existing | incoming
	}
	return existing | incoming
}

func (w *Watcher) stopTimer() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
}
