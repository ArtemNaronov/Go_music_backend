package service

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/library"
	"github.com/temic/go-music/internal/metadata"
	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
)

type stubIndexer struct{}

func (stubIndexer) Scan(context.Context) (models.ScanResult, error) {
	return models.ScanResult{TracksFound: 0}, nil
}

func TestLibraryService_ListTracksPaginationAndSearch(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{ID: "1", Path: `D:\Music\Meteora\01.mp3`, Title: "Foreword", Artist: "Linkin Park", Album: "Meteora"})
	store.Upsert(ctx, models.Track{ID: "2", Path: `D:\Music\Meteora\02.mp3`, Title: "Don't Stay", Artist: "Linkin Park", Album: "Meteora"})
	store.Upsert(ctx, models.Track{ID: "3", Path: `D:\Music\Hits\01.mp3`, Title: "Bohemian Rhapsody", Artist: "Queen", Album: "Greatest Hits"})

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())

	page := svc.ListTracks(ctx, TrackListQuery{Page: 1, Limit: 2, Search: ""})
	if len(page.Items) != 2 || page.Total != 3 || page.TotalPages != 2 {
		t.Fatalf("unexpected page: %+v", page)
	}

	page = svc.ListTracks(ctx, TrackListQuery{Page: 2, Limit: 2})
	if len(page.Items) != 1 {
		t.Fatalf("page 2 items = %d, want 1", len(page.Items))
	}

	search := svc.ListTracks(ctx, TrackListQuery{Page: 1, Limit: 50, Search: "queen"})
	if len(search.Items) != 1 || search.Items[0].Artist != "Queen" {
		t.Fatalf("unexpected search result: %+v", search.Items)
	}

	empty := svc.ListTracks(ctx, TrackListQuery{Page: 99, Limit: 50})
	if len(empty.Items) != 0 || empty.Total != 3 {
		t.Fatalf("unexpected empty page: %+v", empty)
	}
}

func TestListAlbumsPageWithDisabledMetadataDoesNotPanic(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	path := `D:\Music\Queen\Greatest Hits\01.mp3`
	store.Upsert(ctx, models.Track{
		ID: id.Track(path), Path: path,
		Title: "Song", Artist: "Queen", Album: "Greatest Hits",
	})

	var metaSvc *metadata.Service
	svc := NewLibraryService(store, store, stubIndexer{}, metaSvc, zerolog.Nop())

	artistID := store.TrackArtistID(models.Track{Path: path})
	page := svc.ListAlbumsPage(ctx, AlbumListQuery{ArtistID: artistID, Limit: 50})
	if page.Total != 1 {
		t.Fatalf("total = %d, want 1", page.Total)
	}
}
