package metadata

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"

	metacover "github.com/temic/go-music/internal/metadata/cover"
	"github.com/temic/go-music/internal/metadata/model"
	"github.com/temic/go-music/internal/metadata/store"
	"github.com/temic/go-music/internal/metadata/worker"
	"github.com/temic/go-music/internal/models"
)

// Service orchestrates metadata enrichment and API enrichment.
type Service struct {
	store      *store.Store
	provider   Provider
	covers     *metacover.Downloader
	queue      *worker.Queue
	worker     *worker.Worker
	logger     zerolog.Logger
	enabled    bool
	trackAlbum func(models.Track) string
	folderMeta func(models.Track) (artist, album string)
}

// Config configures the metadata service.
type Config struct {
	Enabled    bool
	DataPath   string
	UserAgent  string
	Provider   Provider
	TrackAlbum func(models.Track) string
	FolderMeta func(models.Track) (artist, album string)
}

// NewService creates a metadata enrichment service.
func NewService(cfg Config, logger zerolog.Logger) (*Service, error) {
	if cfg.TrackAlbum == nil || cfg.FolderMeta == nil {
		return nil, fmt.Errorf("metadata service requires track album resolvers")
	}

	dbPath := filepath.Join(cfg.DataPath, "library.db")
	metaStore, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}

	coversDir := filepath.Join(cfg.DataPath, "covers")
	coverDownloader := metacover.NewDownloader(coversDir)
	if err := coverDownloader.EnsureDir(); err != nil {
		_ = metaStore.Close()
		return nil, err
	}

	provider := cfg.Provider
	if provider == nil {
		return nil, fmt.Errorf("metadata provider is required")
	}

	queue := worker.NewQueue(128)
	svc := &Service{
		store:      metaStore,
		provider:   provider,
		covers:     coverDownloader,
		queue:      queue,
		logger:     logger.With().Str("component", "metadata").Logger(),
		enabled:    cfg.Enabled,
		trackAlbum: cfg.TrackAlbum,
		folderMeta: cfg.FolderMeta,
	}
	svc.worker = worker.New(queue, svc, svc.logger)

	return svc, nil
}

// Start launches the background worker.
func (s *Service) Start(ctx context.Context) {
	if !s.enabled {
		s.logger.Info().Msg("metadata enrichment disabled")
		return
	}
	go s.worker.Run(ctx)
	s.logger.Info().Msg("metadata worker started")
}

// Close releases metadata resources.
func (s *Service) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

// OnTrackIndexed enqueues metadata enrichment for the track's album.
func (s *Service) OnTrackIndexed(ctx context.Context, track models.Track) {
	if !s.enabled {
		return
	}

	albumID := s.trackAlbum(track)
	artist, album := s.folderMeta(track)
	if albumID == "" || album == "" {
		return
	}

	shouldFetch, err := s.store.ShouldFetch(ctx, albumID)
	if err != nil {
		s.logger.Warn().Err(err).Str("album_id", albumID).Msg("metadata fetch check failed")
		return
	}
	if !shouldFetch {
		return
	}

	if err := s.store.MarkPending(ctx, albumID, artist, album, track.Year); err != nil {
		s.logger.Warn().Err(err).Str("album_id", albumID).Msg("metadata mark pending failed")
		return
	}

	s.queue.Enqueue(model.Job{
		AlbumID: albumID,
		Artist:  artist,
		Album:   album,
		Year:    track.Year,
	})
}

// ProcessJob fetches and stores metadata for one album.
func (s *Service) ProcessJob(ctx context.Context, job model.Job) error {
	shouldFetch, err := s.store.ShouldFetch(ctx, job.AlbumID)
	if err != nil {
		return err
	}
	if !shouldFetch {
		return nil
	}

	query := model.SearchQuery{
		AlbumID: job.AlbumID,
		Artist:  job.Artist,
		Album:   job.Album,
		Year:    job.Year,
	}

	result, err := s.provider.SearchAlbum(ctx, query)
	if err != nil {
		return s.saveFailure(ctx, job, err)
	}
	if result == nil {
		return s.saveSkipped(ctx, job)
	}

	record := model.AlbumRecord{
		AlbumID:       job.AlbumID,
		Artist:        firstNonEmpty(result.Artist, job.Artist),
		Title:         firstNonEmpty(result.Title, job.Album),
		Year:          pickYear(result.Year, job.Year),
		Genres:        result.Genres,
		Description:   result.Description,
		MusicBrainzID: result.MusicBrainzID,
		FetchStatus:   model.StatusSuccess,
		FetchedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if result.CoverImageURL != "" {
		coverPath, err := s.covers.Download(job.AlbumID, result.CoverImageURL)
		if err != nil {
			s.logger.Warn().Err(err).Str("album_id", job.AlbumID).Msg("cover download failed")
		} else {
			record.CoverPath = coverPath
		}
	}

	return s.store.Save(ctx, record)
}

func (s *Service) saveFailure(ctx context.Context, job model.Job, cause error) error {
	record := model.AlbumRecord{
		AlbumID:     job.AlbumID,
		Artist:      job.Artist,
		Title:       job.Album,
		Year:        job.Year,
		Genres:      []string{},
		FetchStatus: model.StatusFailed,
		UpdatedAt:   time.Now().UTC(),
	}
	if err := s.store.Save(ctx, record); err != nil {
		return err
	}
	return cause
}

func (s *Service) saveSkipped(ctx context.Context, job model.Job) error {
	return s.store.Save(ctx, model.AlbumRecord{
		AlbumID:     job.AlbumID,
		Artist:      job.Artist,
		Title:       job.Album,
		Year:        job.Year,
		Genres:      []string{},
		FetchStatus: model.StatusSkipped,
		FetchedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	})
}

// EnrichAlbum merges stored metadata into an API album.
func (s *Service) EnrichAlbum(ctx context.Context, album models.Album) models.Album {
	if s.store == nil {
		return album
	}

	record, ok, err := s.store.Get(ctx, album.ID)
	if err != nil {
		s.logger.Warn().Err(err).Str("album_id", album.ID).Msg("metadata read failed")
		return album
	}

	return mergeAlbum(album, record, ok)
}

// EnrichAlbums merges metadata into a list of albums.
func (s *Service) EnrichAlbums(ctx context.Context, albums []models.Album) []models.Album {
	if len(albums) == 0 {
		return albums
	}

	out := make([]models.Album, len(albums))
	for i, album := range albums {
		out[i] = s.EnrichAlbum(ctx, album)
	}
	return out
}

// AlbumCoverPath returns a downloaded cover path if available.
func (s *Service) AlbumCoverPath(ctx context.Context, albumID string) (string, bool) {
	if s.store == nil {
		return "", false
	}

	record, ok, err := s.store.Get(ctx, albumID)
	if err != nil || !ok || strings.TrimSpace(record.CoverPath) == "" {
		return "", false
	}
	if _, err := os.Stat(record.CoverPath); err != nil {
		return "", false
	}
	return record.CoverPath, true
}

// AlbumCoverBytes returns downloaded cover bytes for an album.
func (s *Service) AlbumCoverBytes(ctx context.Context, albumID string) ([]byte, bool) {
	if path, ok := s.AlbumCoverPath(ctx, albumID); ok {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return data, true
		}
	}

	data, ok, err := s.covers.Read(albumID)
	if err != nil || !ok {
		return nil, false
	}
	return data, true
}

func mergeAlbum(album models.Album, record model.AlbumRecord, hasRecord bool) models.Album {
	if !hasRecord {
		if album.HasCover {
			album.CoverURL = albumCoverURL(album.ID)
		}
		return album
	}

	album.MetadataStatus = string(record.FetchStatus)
	if record.MusicBrainzID != "" {
		album.MusicBrainzID = record.MusicBrainzID
	}
	if record.Description != "" {
		album.Description = record.Description
	}
	if len(record.Genres) > 0 {
		album.Genres = append([]string(nil), record.Genres...)
	}
	if record.Year > 0 {
		album.Year = record.Year
	}
	if record.CoverPath != "" {
		album.HasCover = true
		album.CoverURL = albumCoverURL(album.ID)
	} else if album.HasCover {
		album.CoverURL = albumCoverURL(album.ID)
	}

	return album
}

func albumCoverURL(albumID string) string {
	return "/api/albums/" + albumID + "/cover"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func pickYear(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
