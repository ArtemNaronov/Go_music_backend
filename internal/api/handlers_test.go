package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/models"
)

type albumTracksStub struct {
	stubLibrary
	album models.Album
}

func (s albumTracksStub) GetAlbum(context.Context, string) (models.Album, bool) {
	return s.album, true
}

func (s albumTracksStub) ListAlbumTracks(context.Context, string) ([]models.Track, bool) {
	return []models.Track{
		{ID: "1", Title: "Track 1", Artist: "Queen", Album: "Greatest Hits"},
	}, true
}

func TestListAlbumTracks(t *testing.T) {
	handler := NewHandler(albumTracksStub{
		album: models.Album{ID: "album-1", Title: "Greatest Hits", Artist: "Queen"},
	}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/api/albums/album-1/tracks", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var tracks []models.Track
	if err := json.Unmarshal(rec.Body.Bytes(), &tracks); err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].Title != "Track 1" {
		t.Fatalf("unexpected tracks: %+v", tracks)
	}
}

func TestListAlbumTracksNotFound(t *testing.T) {
	handler := NewHandler(stubLibrary{}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/api/albums/missing/tracks", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
