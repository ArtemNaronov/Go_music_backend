package service

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/library"
	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
)

func TestListAlbumsPageFilterByArtist(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\Queen\Greatest Hits\01.mp3`), Path: `D:\Music\Queen\Greatest Hits\01.mp3`,
		Title: "Song", Artist: "Queen", Album: "Greatest Hits",
	})
	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\Beatles\Abbey Road\01.mp3`), Path: `D:\Music\Beatles\Abbey Road\01.mp3`,
		Title: "Song", Artist: "The Beatles", Album: "Abbey Road",
	})

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())
	queenID := id.Artist("Queen")

	page := svc.ListAlbumsPage(ctx, AlbumListQuery{ArtistID: queenID, Limit: 50})
	if page.Total != 1 {
		t.Fatalf("total = %d, want 1", page.Total)
	}
	if page.Items[0].Artist != "Queen" {
		t.Fatalf("artist = %q, want Queen", page.Items[0].Artist)
	}
}
