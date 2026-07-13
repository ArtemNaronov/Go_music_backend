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

type searchStub struct {
	stubLibrary
}

func (searchStub) Search(context.Context, service.SearchQuery) models.SearchResult {
	return models.SearchResult{
		Query: "fire",
		Artists: []models.Artist{
			{ID: "artist-1", Name: "Kings of Leon", TrackCount: 1},
		},
		Tracks: []models.Track{
			{ID: "track-1", Title: "Sex on Fire", Artist: "Kings of Leon"},
		},
	}
}

func (searchStub) ListStations(context.Context) []models.Station {
	return []models.Station{
		{ID: "all", Name: "Всё радио", Kind: "all", TrackCount: 10},
		{ID: "genre:rock", Name: "Rock", Kind: "genre", TrackCount: 5},
	}
}

func TestSearchEndpoint(t *testing.T) {
	handler := NewHandler(searchStub{}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=fire", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result models.SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Query != "fire" || len(result.Artists) != 1 || len(result.Tracks) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestListStationsEndpoint(t *testing.T) {
	handler := NewHandler(searchStub{}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/api/stations", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Items []models.Station `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 || body.Items[0].ID != "all" {
		t.Fatalf("unexpected stations: %+v", body.Items)
	}
}

type artistRadioStub struct {
	stubLibrary
}

func (artistRadioStub) Radio(_ context.Context, query service.RadioQuery) (models.RadioQueue, bool) {
	if query.ArtistID != "artist-1" {
		return models.RadioQueue{}, false
	}
	return models.RadioQueue{
		Items: []models.Track{
			{ID: "track-1", Title: "Track", Artist: "Queen"},
		},
		TotalAvailable: 1,
		Returned:       1,
	}, true
}

func TestRadioArtistIDEndpoint(t *testing.T) {
	handler := NewHandler(artistRadioStub{}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/api/radio?artist_id=artist-1", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
