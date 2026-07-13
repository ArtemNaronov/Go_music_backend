package library

import (
	"context"
	"testing"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
)

func TestGetAlbumDoesNotRebuildAllAlbums(t *testing.T) {
	store := NewStore(`D:\Music`)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		path := `D:\Music\Artist\Album` + string(rune('0'+i)) + `\track.mp3`
		store.Upsert(ctx, models.Track{
			ID:     id.Track(path),
			Path:   path,
			Title:  "Track",
			Artist: "Artist",
			Album:  "Album",
		})
	}

	targetPath := `D:\Music\Queen\Greatest Hits\01.mp3`
	targetID := store.TrackAlbumID(models.Track{Path: targetPath})
	store.Upsert(ctx, models.Track{
		ID:       id.Track(targetPath),
		Path:     targetPath,
		Title:    "Bohemian Rhapsody",
		Artist:   "Queen",
		Album:    "Greatest Hits",
		Duration: 355,
		HasCover: true,
		Year:     1981,
	})

	album, ok := store.GetAlbum(ctx, targetID)
	if !ok {
		t.Fatal("expected album")
	}
	if album.Title != "Greatest Hits" || album.Artist != "Queen" {
		t.Fatalf("unexpected album: %+v", album)
	}
	if album.TrackCount != 1 || album.Duration != 355 || !album.HasCover {
		t.Fatalf("unexpected album stats: %+v", album)
	}
}

func TestAlbumIndexMaintainedOnRemove(t *testing.T) {
	store := NewStore(`D:\Music`)
	ctx := context.Background()

	first := models.Track{
		ID:     id.Track(`D:\Music\Queen\Album\01.mp3`),
		Path:   `D:\Music\Queen\Album\01.mp3`,
		Title:  "One",
		Artist: "Queen",
		Album:  "Album",
	}
	second := models.Track{
		ID:     id.Track(`D:\Music\Queen\Album\02.mp3`),
		Path:   `D:\Music\Queen\Album\02.mp3`,
		Title:  "Two",
		Artist: "Queen",
		Album:  "Album",
	}

	store.Upsert(ctx, first)
	store.Upsert(ctx, second)

	albumID := store.TrackAlbumID(first)
	if tracks := store.ListAlbumTracks(ctx, albumID); len(tracks) != 2 {
		t.Fatalf("album tracks = %d, want 2", len(tracks))
	}

	if !store.Remove(ctx, first.ID) {
		t.Fatal("expected remove to succeed")
	}

	if tracks := store.ListAlbumTracks(ctx, albumID); len(tracks) != 1 {
		t.Fatalf("album tracks after remove = %d, want 1", len(tracks))
	}
}
