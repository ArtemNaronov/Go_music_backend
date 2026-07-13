package library

import (
	"context"
	"testing"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
)

func TestStore_UpsertAndQuery(t *testing.T) {
	store := NewStore(`D:\Music`)
	ctx := context.Background()

	track := models.Track{
		ID:       id.Track(`D:\Music\Queen\Greatest Hits\01.mp3`),
		Path:     `D:\Music\Queen\Greatest Hits\01.mp3`,
		Title:    "Bohemian Rhapsody",
		Artist:   "Queen",
		Album:    "Greatest Hits",
		Duration: 355.4,
		HasCover: true,
	}

	store.Upsert(ctx, track)

	got, ok := store.GetTrack(ctx, track.ID)
	if !ok {
		t.Fatal("expected track by id")
	}
	if got.Title != track.Title {
		t.Fatalf("title = %q, want %q", got.Title, track.Title)
	}

	got, ok = store.GetTrackByPath(ctx, track.Path)
	if !ok {
		t.Fatal("expected track by path")
	}
	if got.ID != track.ID {
		t.Fatalf("id = %q, want %q", got.ID, track.ID)
	}

	if store.Count(ctx) != 1 {
		t.Fatalf("count = %d, want 1", store.Count(ctx))
	}

	artists := store.ListArtists(ctx)
	if len(artists) != 1 || artists[0].Name != "Queen" {
		t.Fatalf("unexpected artists: %+v", artists)
	}

	albums := store.ListAlbums(ctx)
	if len(albums) != 1 || albums[0].Title != "Greatest Hits" {
		t.Fatalf("unexpected albums: %+v", albums)
	}

	tracks := store.ListAlbumTracks(ctx, albums[0].ID)
	if len(tracks) != 1 {
		t.Fatalf("album tracks = %d, want 1", len(tracks))
	}
}

func TestStore_RemoveAndReplaceAll(t *testing.T) {
	store := NewStore(`D:\Music`)
	ctx := context.Background()

	first := models.Track{
		ID:     id.Track(`D:\Music\A\a.mp3`),
		Path:   `D:\Music\A\a.mp3`,
		Title:  "A",
		Artist: "Artist",
		Album:  "Album",
	}
	second := models.Track{
		ID:     id.Track(`D:\Music\B\b.mp3`),
		Path:   `D:\Music\B\b.mp3`,
		Title:  "B",
		Artist: "Artist",
		Album:  "Album",
	}

	store.Upsert(ctx, first)
	store.Upsert(ctx, second)

	if !store.Remove(ctx, first.ID) {
		t.Fatal("expected remove to succeed")
	}
	if store.Count(ctx) != 1 {
		t.Fatalf("count = %d, want 1", store.Count(ctx))
	}

	store.ReplaceAll(ctx, []models.Track{second})
	if store.Count(ctx) != 1 {
		t.Fatalf("count after replace = %d, want 1", store.Count(ctx))
	}
	if _, ok := store.GetTrack(ctx, first.ID); ok {
		t.Fatal("old track should be gone after replace")
	}
}

func TestStore_UpsertPathCaseInsensitive(t *testing.T) {
	store := NewStore(`D:\Music`)
	ctx := context.Background()

	track := models.Track{
		ID:     id.Track(`D:\Music\Song.mp3`),
		Path:   `D:\Music\Song.mp3`,
		Title:  "Song",
		Artist: "Artist",
		Album:  "Album",
	}

	store.Upsert(ctx, track)

	if !store.RemoveByPath(ctx, `d:\music\song.mp3`) {
		t.Fatal("expected case-insensitive path removal")
	}
}
