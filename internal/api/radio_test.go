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

type radioStub struct {
	stubLibrary
}

func (radioStub) Radio(context.Context, service.RadioQuery) (models.RadioQueue, bool) {
	return models.RadioQueue{
		Items: []models.Track{
			{ID: "1", Title: "Random Song", Artist: "Artist", Album: "Album"},
		},
		TotalAvailable: 10,
		Returned:       1,
	}, true
}

func TestRadioEndpoint(t *testing.T) {
	handler := NewHandler(radioStub{}, zerolog.Nop())
	router := NewRouter("secret", handler, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/api/radio?limit=1", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var queue models.RadioQueue
	if err := json.Unmarshal(rec.Body.Bytes(), &queue); err != nil {
		t.Fatal(err)
	}
	if queue.Returned != 1 || queue.Items[0].Title != "Random Song" {
		t.Fatalf("unexpected queue: %+v", queue)
	}
}
