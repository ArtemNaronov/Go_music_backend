package musicbrainz_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/temic/go-music/internal/metadata/model"
	"github.com/temic/go-music/internal/metadata/musicbrainz"
)

func TestProviderSearchAlbum(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ws/2/release" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"releases": []map[string]any{
				{
					"id":    "release-id",
					"title": "The Last Stand",
					"date":  "2016-08-19",
					"artist-credit": []map[string]any{
						{
							"name": "Sabaton",
							"artist": map[string]string{
								"name": "Sabaton",
							},
						},
					},
					"tags": []map[string]string{
						{"name": "power metal"},
					},
					"disambiguation": "2016 album",
				},
			},
		})
	}))
	defer server.Close()

	provider := musicbrainz.NewWithAPIBase(server.URL+"/ws/2", "test-agent")
	result, err := provider.SearchAlbum(context.Background(), model.SearchQuery{
		AlbumID: "album-1",
		Artist:  "Sabaton",
		Album:   "The Last Stand",
		Year:    2016,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if result.MusicBrainzID != "release-id" {
		t.Fatalf("mbid = %q", result.MusicBrainzID)
	}
	if result.Year != 2016 {
		t.Fatalf("year = %d", result.Year)
	}
	if len(result.Genres) != 1 || result.Genres[0] != "power metal" {
		t.Fatalf("genres = %+v", result.Genres)
	}
	if result.CoverImageURL == "" {
		t.Fatal("expected cover url")
	}
}
