package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/api"
	"github.com/temic/go-music/internal/config"
	"github.com/temic/go-music/internal/library"
	"github.com/temic/go-music/internal/metadata"
	"github.com/temic/go-music/internal/metadata/musicbrainz"
	"github.com/temic/go-music/internal/scanner"
	"github.com/temic/go-music/internal/service"
)

// Server runs the music HTTP API with library scanning and watching.
type Server struct {
	cfg      config.Config
	logger   zerolog.Logger
	dataPath string

	mu       sync.Mutex
	running  bool
	cancel   context.CancelFunc
	http     *http.Server
	store    *library.Store
	metadata *metadata.Service
}

// New creates a server instance from configuration.
func New(cfg config.Config, logger zerolog.Logger) *Server {
	cfg = config.Normalize(cfg)

	return &Server{
		cfg:      cfg,
		logger:   logger.With().Str("component", "app").Logger(),
		dataPath: config.ResolveDataPath(cfg.DataPath),
	}
}

// IsRunning reports whether the server is currently listening.
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// TrackCount returns the number of indexed tracks.
func (s *Server) TrackCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store == nil {
		return 0
	}
	return s.store.Count(context.Background())
}

// Addr returns the listen address from config.
func (s *Server) Addr() string {
	return s.cfg.Addr()
}

// MusicPath returns the current music directory.
func (s *Server) MusicPath() string {
	return s.cfg.MusicPath
}

// Start scans the library and begins listening for HTTP requests.
func (s *Server) Start(parent context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.mu.Unlock()

	info, err := os.Stat(s.cfg.MusicPath)
	if err != nil {
		return fmt.Errorf("music path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("music path is not a directory: %s", s.cfg.MusicPath)
	}

	if err := os.MkdirAll(s.dataPath, 0o755); err != nil {
		return fmt.Errorf("data path: %w", err)
	}

	store := library.NewStore(s.cfg.MusicPath)
	sc := scanner.New(s.cfg.MusicPath, store, store, s.logger)

	var metaSvc *metadata.Service
	if s.cfg.Metadata.Enabled {
		metaSvc, err = metadata.NewService(metadata.Config{
			Enabled:    true,
			DataPath:   s.dataPath,
			UserAgent:  s.cfg.Metadata.UserAgent,
			Provider:   musicbrainz.New(s.cfg.Metadata.UserAgent),
			TrackAlbum: store.TrackAlbumID,
			FolderMeta: store.TrackFolderAlbum,
		}, s.logger)
		if err != nil {
			return fmt.Errorf("metadata service: %w", err)
		}
		sc.SetIndexHook(metaSvc)
	}

	var enricher service.AlbumEnricher
	if metaSvc != nil {
		enricher = metaSvc
	}

	svc := service.NewLibraryService(store, store, sc, enricher, s.logger)

	ctx, cancel := context.WithCancel(parent)
	if metaSvc != nil {
		metaSvc.Start(ctx)
	}

	s.logger.Info().Str("music_path", s.cfg.MusicPath).Str("data_path", s.dataPath).Msg("scanning library")
	if _, err := svc.Rescan(ctx); err != nil {
		cancel()
		if metaSvc != nil {
			_ = metaSvc.Close()
		}
		return fmt.Errorf("initial scan: %w", err)
	}

	watcher := scanner.NewWatcher(s.cfg.MusicPath, sc, s.logger)
	go func() {
		if err := watcher.Run(ctx); err != nil {
			s.logger.Error().Err(err).Msg("filesystem watcher stopped with error")
		}
	}()

	handler := api.NewHandler(svc, s.logger)
	router := api.NewRouter(s.cfg.Token, handler, s.logger)

	httpServer := &http.Server{
		Addr:         s.cfg.Addr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	s.mu.Lock()
	s.running = true
	s.cancel = cancel
	s.http = httpServer
	s.store = store
	s.metadata = metaSvc
	s.mu.Unlock()

	go func() {
		s.logger.Info().
			Str("music_path", s.cfg.MusicPath).
			Str("data_path", filepath.Clean(s.dataPath)).
			Str("addr", s.cfg.Addr()).
			Int("tracks", store.Count(ctx)).
			Msg("http server started")

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error().Err(err).Msg("http server failed")
			_ = s.Stop()
		}
	}()

	return nil
}

// Stop gracefully shuts down the HTTP server and watcher.
func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}

	cancel := s.cancel
	httpServer := s.http
	metaSvc := s.metadata
	s.running = false
	s.cancel = nil
	s.http = nil
	s.store = nil
	s.metadata = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if metaSvc != nil {
		if err := metaSvc.Close(); err != nil {
			s.logger.Warn().Err(err).Msg("metadata service close failed")
		}
	}

	if httpServer == nil {
		return nil
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	s.logger.Info().Msg("shutting down music server")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}
