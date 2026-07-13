package library

import (
	"context"
	"testing"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
)

func TestListAlbumsGroupsByFolderNotTags(t *testing.T) {
	root := `D:\Music`
	store := NewStore(root)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{
		ID:     id.Track(`D:\Music\Queen\Greatest Hits\01.mp3`),
		Path:   `D:\Music\Queen\Greatest Hits\01.mp3`,
		Title:  "We Will Rock You",
		Artist: "Queen",
		Album:  "Greatest Hits",
	})
	store.Upsert(ctx, models.Track{
		ID:     id.Track(`D:\Music\Queen\Greatest Hits\02.mp3`),
		Path:   `D:\Music\Queen\Greatest Hits\02.mp3`,
		Title:  "We Are the Champions",
		Artist: "QUEEN",
		Album:  "Greatest Hits Vol. 1",
	})

	albums := store.ListAlbums(ctx)
	if len(albums) != 1 {
		t.Fatalf("albums = %d, want 1", len(albums))
	}
	if albums[0].Title != "Greatest Hits" {
		t.Fatalf("album title = %q, want Greatest Hits", albums[0].Title)
	}
	if albums[0].TrackCount != 2 {
		t.Fatalf("track_count = %d, want 2", albums[0].TrackCount)
	}
}
