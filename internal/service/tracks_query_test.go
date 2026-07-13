package service

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/library"
	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
)

func TestListTracksFiltersAndSort(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{
		ID:          id.Track(`D:\Music\Queen\Greatest Hits\02.mp3`),
		Path:        `D:\Music\Queen\Greatest Hits\02.mp3`,
		Title:       "We Will Rock You",
		Artist:      "Queen",
		Album:       "Greatest Hits",
		TrackNumber: 2,
		Duration:    120,
	})
	store.Upsert(ctx, models.Track{
		ID:          id.Track(`D:\Music\Queen\Greatest Hits\01.mp3`),
		Path:        `D:\Music\Queen\Greatest Hits\01.mp3`,
		Title:       "Bohemian Rhapsody",
		Artist:      "Queen",
		Album:       "Greatest Hits",
		TrackNumber: 1,
		Duration:    354,
	})
	store.Upsert(ctx, models.Track{
		ID:     id.Track(`D:\Music\Beatles\Abbey Road\01.mp3`),
		Path:   `D:\Music\Beatles\Abbey Road\01.mp3`,
		Title:  "Come Together",
		Artist: "The Beatles",
		Album:  "Abbey Road",
	})

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())
	albumID := id.Album("Queen", "Greatest Hits")

	page := svc.ListTracks(ctx, TrackListQuery{
		AlbumID: albumID,
		Sort:    "track_number",
		Limit:   50,
	})
	if page.Total != 2 {
		t.Fatalf("total = %d, want 2", page.Total)
	}
	if page.Items[0].Title != "Bohemian Rhapsody" {
		t.Fatalf("first track = %q", page.Items[0].Title)
	}

	artistID := store.TrackArtistID(models.Track{Path: `D:\Music\Queen\Greatest Hits\01.mp3`})
	page = svc.ListTracks(ctx, TrackListQuery{ArtistID: artistID, Limit: 50})
	if page.Total != 2 {
		t.Fatalf("artist total = %d, want 2", page.Total)
	}
}

func TestListArtistsPage(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\A\Album\01.mp3`), Path: `D:\Music\A\Album\01.mp3`,
		Title: "One", Artist: "A", Album: "Album",
	})
	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\B\Album\01.mp3`), Path: `D:\Music\B\Album\01.mp3`,
		Title: "Two", Artist: "B", Album: "Album",
	})

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())
	page := svc.ListArtistsPage(ctx, 1, 1)
	if page.Total != 2 || len(page.Items) != 1 || page.TotalPages != 2 {
		t.Fatalf("unexpected artists page: %+v", page)
	}
}
