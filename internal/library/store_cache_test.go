package library

import (
	"context"
	"testing"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
)

func TestListCacheInvalidatedOnMutation(t *testing.T) {
	store := NewStore(`D:\Music`)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\A\Album\01.mp3`), Path: `D:\Music\A\Album\01.mp3`,
		Title: "One", Artist: "A", Album: "Album",
	})

	first := store.ListArtists(ctx)
	if len(first) != 1 || first[0].Name != "A" {
		t.Fatalf("unexpected artists: %+v", first)
	}

	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\B\Album\01.mp3`), Path: `D:\Music\B\Album\01.mp3`,
		Title: "Two", Artist: "B", Album: "Album",
	})

	second := store.ListArtists(ctx)
	if len(second) != 2 {
		t.Fatalf("artists after upsert = %d, want 2", len(second))
	}
}

func TestListTracksByArtistUsesIndex(t *testing.T) {
	store := NewStore(`D:\Music`)
	ctx := context.Background()

	queenPath := `D:\Music\Queen\Greatest Hits\01.mp3`
	beatlesPath := `D:\Music\Beatles\Abbey Road\01.mp3`

	store.Upsert(ctx, models.Track{
		ID: id.Track(queenPath), Path: queenPath,
		Title: "Queen Song", Artist: "Queen", Album: "Greatest Hits",
	})
	store.Upsert(ctx, models.Track{
		ID: id.Track(beatlesPath), Path: beatlesPath,
		Title: "Beatles Song", Artist: "The Beatles", Album: "Abbey Road",
	})

	queenID := store.TrackArtistID(models.Track{Path: queenPath})
	tracks := store.ListTracksByArtist(ctx, queenID)
	if len(tracks) != 1 || tracks[0].Title != "Queen Song" {
		t.Fatalf("unexpected queen tracks: %+v", tracks)
	}

	albums := store.ListAlbumsByArtist(ctx, queenID)
	if len(albums) != 1 || albums[0].Title != "Greatest Hits" {
		t.Fatalf("unexpected queen albums: %+v", albums)
	}
}

func TestGetAlbumReadsCachedMeta(t *testing.T) {
	store := NewStore(`D:\Music`)
	ctx := context.Background()

	path := `D:\Music\Queen\Greatest Hits\01.mp3`
	store.Upsert(ctx, models.Track{
		ID: id.Track(path), Path: path,
		Title: "Song", Artist: "Queen", Album: "Greatest Hits",
		Duration: 100, HasCover: true, Year: 1981,
	})

	albumID := store.TrackAlbumID(models.Track{Path: path})
	album, ok := store.GetAlbum(ctx, albumID)
	if !ok {
		t.Fatal("expected album")
	}
	if album.TrackCount != 1 || album.Duration != 100 || !album.HasCover || album.Year != 1981 {
		t.Fatalf("unexpected album meta: %+v", album)
	}
}
