package service

import (
	"context"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/cover"
	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/internal/repository"
)

const (
	defaultPage  = 1
	defaultLimit = 50
	maxLimit     = 200
)

// Indexer performs full and incremental library indexing.
type Indexer interface {
	Scan(ctx context.Context) (models.ScanResult, error)
}

// LibraryService implements read operations and library maintenance.
type LibraryService struct {
	library  repository.Library
	covers   repository.CoverStore
	indexer  Indexer
	metadata AlbumEnricher
	logger   zerolog.Logger
	scanMu   sync.Mutex
}

// NewLibraryService creates a library service with injected dependencies.
func NewLibraryService(
	library repository.Library,
	covers repository.CoverStore,
	indexer Indexer,
	metadata AlbumEnricher,
	logger zerolog.Logger,
) *LibraryService {
	return &LibraryService{
		library:  library,
		covers:   covers,
		indexer:  indexer,
		metadata: normalizeEnricher(metadata),
		logger:   logger.With().Str("component", "service").Logger(),
	}
}

func (s *LibraryService) GetTrack(ctx context.Context, id string) (models.Track, bool) {
	return s.library.GetTrack(ctx, id)
}

func (s *LibraryService) ListArtists(ctx context.Context) []models.Artist {
	return s.ListArtistsPage(ctx, 1, maxLimit).Items
}

func (s *LibraryService) ListAlbums(ctx context.Context) []models.Album {
	return s.ListAlbumsPage(ctx, AlbumListQuery{Page: 1, Limit: maxLimit}).Items
}

func (s *LibraryService) GetAlbum(ctx context.Context, albumID string) (models.Album, bool) {
	album, ok := s.library.GetAlbum(ctx, albumID)
	if !ok {
		return models.Album{}, false
	}
	if hasEnricher(s.metadata) {
		album = s.metadata.EnrichAlbum(ctx, album)
	}
	return album, true
}

func (s *LibraryService) ListAlbumTracks(ctx context.Context, albumID string) ([]models.Track, bool) {
	if _, ok := s.library.GetAlbum(ctx, albumID); !ok {
		return nil, false
	}
	return s.library.ListAlbumTracks(ctx, albumID), true
}

func (s *LibraryService) GetCover(ctx context.Context, trackID string) ([]byte, string, bool) {
	return s.covers.GetCover(ctx, trackID)
}

// ResolveCover returns embedded artwork or a folder image fallback.
func (s *LibraryService) ResolveCover(ctx context.Context, trackID string) ([]byte, string, bool) {
	if data, mime, ok := s.covers.GetCover(ctx, trackID); ok {
		return data, mime, true
	}

	track, ok := s.library.GetTrack(ctx, trackID)
	if !ok {
		return nil, "", false
	}

	return cover.FromFolder(track.Path)
}

func (s *LibraryService) Rescan(ctx context.Context) (models.ScanResult, error) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	s.logger.Info().Msg("rescan requested")

	result, err := s.indexer.Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("rescan failed")
		return models.ScanResult{}, err
	}

	s.logger.Info().
		Int("tracks", result.TracksFound).
		Float64("duration_seconds", result.Duration).
		Msg("rescan completed")

	return result, nil
}

func normalizePagination(page, limit int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

func calcTotalPages(total, limit int) int {
	if total == 0 {
		return 0
	}
	return (total + limit - 1) / limit
}

func filterTracks(tracks []models.Track, search string) []models.Track {
	query := strings.ToLower(strings.TrimSpace(search))
	if query == "" {
		return tracks
	}

	filtered := make([]models.Track, 0, len(tracks))
	for _, track := range tracks {
		if trackFieldsMatch(track, query) {
			filtered = append(filtered, track)
		}
	}

	return filtered
}
