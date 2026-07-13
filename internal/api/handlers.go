package api

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/internal/service"
	"github.com/temic/go-music/internal/stream"
)

// Library defines the service methods used by HTTP handlers.
type Library interface {
	ListArtistsPage(ctx context.Context, page, limit int) models.ArtistsPage
	ListAlbumsPage(ctx context.Context, query service.AlbumListQuery) models.AlbumsPage
	GetAlbum(ctx context.Context, albumID string) (models.Album, bool)
	ListAlbumTracks(ctx context.Context, albumID string) ([]models.Track, bool)
	ListTracks(ctx context.Context, query service.TrackListQuery) models.TracksPage
	GetTrack(ctx context.Context, id string) (models.Track, bool)
	ResolveCover(ctx context.Context, trackID string) ([]byte, string, bool)
	ResolveAlbumCover(ctx context.Context, albumID string) ([]byte, string, bool)
	Search(ctx context.Context, query service.SearchQuery) models.SearchResult
	ListStations(ctx context.Context) []models.Station
	Radio(ctx context.Context, query service.RadioQuery) (models.RadioQueue, bool)
	Rescan(ctx context.Context) (models.ScanResult, error)
}

// Handler serves the REST API.
type Handler struct {
	library Library
	logger  zerolog.Logger
}

// NewHandler creates an API handler with injected dependencies.
func NewHandler(library Library, logger zerolog.Logger) *Handler {
	return &Handler{
		library: library,
		logger:  logger.With().Str("component", "api").Logger(),
	}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (h *Handler) ListArtists(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.library.ListArtistsPage(
		r.Context(),
		queryInt(r, "page"),
		queryInt(r, "limit"),
	))
}

func (h *Handler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.library.ListAlbumsPage(r.Context(), service.AlbumListQuery{
		Page:     queryInt(r, "page"),
		Limit:    queryInt(r, "limit"),
		ArtistID: r.URL.Query().Get("artist_id"),
	}))
}

func (h *Handler) ListAlbumTracks(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")

	tracks, ok := h.library.ListAlbumTracks(r.Context(), albumID)
	if !ok {
		writeError(w, http.StatusNotFound, "Album not found")
		return
	}

	writeJSON(w, http.StatusOK, tracks)
}

func (h *Handler) CoverAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")

	if _, ok := h.library.GetAlbum(r.Context(), albumID); !ok {
		writeError(w, http.StatusNotFound, "Album not found")
		return
	}

	data, mime, ok := h.library.ResolveAlbumCover(r.Context(), albumID)
	if !ok {
		writeError(w, http.StatusNotFound, "Cover not found")
		return
	}

	writeCover(w, data, mime)
}

func (h *Handler) ListTracks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.library.ListTracks(r.Context(), service.TrackListQuery{
		Page:     queryInt(r, "page"),
		Limit:    queryInt(r, "limit"),
		Search:   r.URL.Query().Get("search"),
		ArtistID: r.URL.Query().Get("artist_id"),
		AlbumID:  r.URL.Query().Get("album_id"),
		Sort:     r.URL.Query().Get("sort"),
	}))
}

func (h *Handler) GetTrack(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")

	track, ok := h.library.GetTrack(r.Context(), trackID)
	if !ok {
		writeError(w, http.StatusNotFound, "Track not found")
		return
	}

	writeJSON(w, http.StatusOK, track)
}

func (h *Handler) StreamTrack(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")

	track, ok := h.library.GetTrack(r.Context(), trackID)
	if !ok {
		writeError(w, http.StatusNotFound, "Track not found")
		return
	}

	if err := stream.FileWithInfo(w, r, stream.FileInfo{
		Path:    track.Path,
		Size:    track.Size,
		ModTime: track.ModifiedAt,
		Format:  track.Format,
	}); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "Track file not found")
			return
		}

		h.logger.Error().Err(err).Str("track_id", trackID).Msg("failed to stream track")
		writeError(w, http.StatusInternalServerError, "Failed to stream track")
	}
}

func (h *Handler) CoverTrack(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")

	if _, ok := h.library.GetTrack(r.Context(), trackID); !ok {
		writeError(w, http.StatusNotFound, "Track not found")
		return
	}

	data, mime, ok := h.library.ResolveCover(r.Context(), trackID)
	if !ok {
		writeError(w, http.StatusNotFound, "Cover not found")
		return
	}

	writeCover(w, data, mime)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.library.Search(r.Context(), service.SearchQuery{
		Query: r.URL.Query().Get("q"),
		Limit: queryInt(r, "limit"),
	}))
}

func (h *Handler) ListStations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"items": h.library.ListStations(r.Context()),
	})
}

func (h *Handler) Radio(w http.ResponseWriter, r *http.Request) {
	queue, ok := h.library.Radio(r.Context(), service.RadioQuery{
		Limit:    queryInt(r, "limit"),
		Exclude:  queryCSV(r, "exclude"),
		ArtistID: r.URL.Query().Get("artist_id"),
		Station:  r.URL.Query().Get("station"),
		Seed:     r.URL.Query().Get("seed"),
	})
	if !ok {
		writeError(w, http.StatusNotFound, "No tracks available for radio")
		return
	}

	writeJSON(w, http.StatusOK, queue)
}

func (h *Handler) Rescan(w http.ResponseWriter, r *http.Request) {
	result, err := h.library.Rescan(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("rescan request failed")
		writeError(w, http.StatusInternalServerError, "Failed to rescan library")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeCover(w http.ResponseWriter, data []byte, mime string) {
	if mime == "" {
		mime = "image/jpeg"
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
