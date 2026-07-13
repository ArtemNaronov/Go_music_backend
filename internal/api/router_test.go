package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/internal/service"
)

type stubLibrary struct{}

func (stubLibrary) ListArtistsPage(context.Context, int, int) models.ArtistsPage {
	return models.ArtistsPage{}
}
func (stubLibrary) ListAlbumsPage(context.Context, service.AlbumListQuery) models.AlbumsPage {
	return models.AlbumsPage{}
}
func (stubLibrary) GetAlbum(context.Context, string) (models.Album, bool) {
	return models.Album{}, false
}
func (stubLibrary) ListAlbumTracks(context.Context, string) ([]models.Track, bool) {
	return nil, false
}
func (stubLibrary) ListTracks(context.Context, service.TrackListQuery) models.TracksPage {
	return models.TracksPage{}
}
func (stubLibrary) GetTrack(context.Context, string) (models.Track, bool) {
	return models.Track{}, false
}
func (stubLibrary) ResolveCover(context.Context, string) ([]byte, string, bool) {
	return nil, "", false
}
func (stubLibrary) ResolveAlbumCover(context.Context, string) ([]byte, string, bool) {
	return nil, "", false
}
func (stubLibrary) Search(context.Context, service.SearchQuery) models.SearchResult {
	return models.SearchResult{}
}
func (stubLibrary) ListStations(context.Context) []models.Station {
	return nil
}
func (stubLibrary) Radio(context.Context, service.RadioQuery) (models.RadioQueue, bool) {
	return models.RadioQueue{}, false
}
func (stubLibrary) Rescan(context.Context) (models.ScanResult, error) {
	return models.ScanResult{}, nil
}

func TestHealthIsPublic(t *testing.T) {
	handler := NewHandler(stubLibrary{}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAPIRequiresBearerToken(t *testing.T) {
	handler := NewHandler(stubLibrary{}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/api/artists", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var body models.APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Error || body.Message == "" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestAPIAcceptsBearerToken(t *testing.T) {
	handler := NewHandler(stubLibrary{}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/api/artists", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAPIAcceptsQueryToken(t *testing.T) {
	handler := NewHandler(stubLibrary{}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/api/artists?token=secret", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
