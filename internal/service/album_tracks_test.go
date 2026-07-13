package service

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/library"
	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
)

func TestLibraryService_ListAlbumTracks(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{
		ID:     id.Track(`D:\Music\Queen\Greatest Hits\01.mp3`),
		Path:   `D:\Music\Queen\Greatest Hits\01.mp3`,
		Title:  "Bohemian Rhapsody",
		Artist: "Queen",
		Album:  "Greatest Hits",
	})
	store.Upsert(ctx, models.Track{
		ID:     id.Track(`D:\Music\Queen\Greatest Hits\02.mp3`),
		Path:   `D:\Music\Queen\Greatest Hits\02.mp3`,
		Title:  "We Will Rock You",
		Artist: "Queen",
		Album:  "Greatest Hits",
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
	tracks, ok := svc.ListAlbumTracks(ctx, albumID)
	if !ok {
		t.Fatal("expected album tracks")
	}
	if len(tracks) != 2 {
		t.Fatalf("track count = %d, want 2", len(tracks))
	}

	if _, ok := svc.ListAlbumTracks(ctx, "missing"); ok {
		t.Fatal("expected missing album to return false")
	}
}
