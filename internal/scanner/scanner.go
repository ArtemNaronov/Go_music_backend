package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/internal/repository"
)

const defaultWorkers = 4

// IndexHook is notified after a track is indexed.
type IndexHook interface {
	OnTrackIndexed(ctx context.Context, track models.Track)
}

// Scanner indexes audio files from a music directory.
type Scanner struct {
	musicPath string
	library   repository.Library
	covers    repository.CoverStore
	logger    zerolog.Logger
	workers   int
	indexHook IndexHook
}

// New creates a scanner with injected dependencies.
func New(
	musicPath string,
	library repository.Library,
	covers repository.CoverStore,
	logger zerolog.Logger,
) *Scanner {
	workers := runtime.NumCPU()
	if workers < defaultWorkers {
		workers = defaultWorkers
	}

	return &Scanner{
		musicPath: musicPath,
		library:   library,
		covers:    covers,
		logger:    logger.With().Str("component", "scanner").Logger(),
		workers:   workers,
	}
}

// SetIndexHook registers a callback for newly indexed tracks.
func (s *Scanner) SetIndexHook(hook IndexHook) {
	s.indexHook = hook
}

func (s *Scanner) notifyIndexed(ctx context.Context, track models.Track) {
	if s.indexHook != nil {
		s.indexHook.OnTrackIndexed(ctx, track)
	}
}

type fileJob struct {
	path   string
	format string
}

type fileResult struct {
	track parsedTrack
	err   error
	path  string
}

// Scan walks the music directory and rebuilds the in-memory library.
func (s *Scanner) Scan(ctx context.Context) (models.ScanResult, error) {
	started := time.Now()

	paths, err := s.collectFiles()
	if err != nil {
		return models.ScanResult{}, err
	}

	s.logger.Info().Int("files", len(paths)).Msg("scan started")

	jobs := make(chan fileJob)
	results := make(chan fileResult)

	var wg sync.WaitGroup
	for range s.workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					return
				}

				parsed, parseErr := parseFileSafe(s.musicPath, job.path, job.format)
				if parseErr == nil {
					parsed.track = enrichFolderCover(parsed.track)
				}

				results <- fileResult{
					track: parsed,
					err:   parseErr,
					path:  job.path,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, path := range paths {
			format, ok := isSupported(path)
			if !ok {
				continue
			}
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- fileJob{path: path, format: format}:
			}
		}
		close(jobs)
	}()

	tracks := make([]models.Track, 0, len(paths))
	var coverMu sync.Mutex
	coverBatch := make(map[string]coverEntry)

	for result := range results {
		if result.err != nil {
			s.logger.Warn().Err(result.err).Str("path", result.path).Msg("failed to parse file")
			continue
		}

		tracks = append(tracks, result.track.track)

		if len(result.track.cover) > 0 {
			coverMu.Lock()
			coverBatch[result.track.track.ID] = coverEntry{
				data: result.track.cover,
				mime: result.track.mime,
			}
			coverMu.Unlock()
		}
	}

	if err := ctx.Err(); err != nil {
		return models.ScanResult{}, err
	}

	s.library.ReplaceAll(ctx, tracks)
	s.covers.ClearCovers(ctx)
	for trackID, cover := range coverBatch {
		s.covers.SetCover(ctx, trackID, cover.data, cover.mime)
	}

	for _, track := range tracks {
		s.notifyIndexed(ctx, track)
	}

	scanResult := models.ScanResult{
		TracksFound: len(tracks),
		Duration:    time.Since(started).Seconds(),
	}

	s.logger.Info().
		Int("tracks", scanResult.TracksFound).
		Float64("duration_seconds", scanResult.Duration).
		Msg("scan completed")

	return scanResult, nil
}

// ScanFile parses and upserts a single file. Used by the filesystem watcher.
func (s *Scanner) ScanFile(ctx context.Context, path string) error {
	format, ok := isSupported(path)
	if !ok {
		return s.removePath(ctx, path)
	}

	parsed, err := parseFile(s.musicPath, path, format)
	if err != nil {
		return err
	}

	parsed.track = enrichFolderCover(parsed.track)
	s.library.Upsert(ctx, parsed.track)

	if len(parsed.cover) > 0 {
		s.covers.SetCover(ctx, parsed.track.ID, parsed.cover, parsed.mime)
	} else {
		s.covers.RemoveCover(ctx, parsed.track.ID)
	}

	s.notifyIndexed(ctx, parsed.track)
	return nil
}

// RemovePath deletes a track from the library by file path.
func (s *Scanner) RemovePath(ctx context.Context, path string) error {
	return s.removePath(ctx, path)
}

func (s *Scanner) removePath(ctx context.Context, path string) error {
	track, ok := s.library.GetTrackByPath(ctx, path)
	if ok {
		s.covers.RemoveCover(ctx, track.ID)
	}
	s.library.RemoveByPath(ctx, path)
	return nil
}

func (s *Scanner) collectFiles() ([]string, error) {
	var paths []string

	err := filepath.WalkDir(s.musicPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			s.logger.Warn().Err(err).Str("path", path).Msg("failed to access path")
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if _, ok := isSupported(path); ok {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func parseFileSafe(musicPath, path, format string) (parsed parsedTrack, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("parse panic: %v", recovered)
		}
	}()
	return parseFile(musicPath, path, format)
}

type coverEntry struct {
	data []byte
	mime string
}
