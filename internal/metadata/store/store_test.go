package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/temic/go-music/internal/metadata/model"
	"github.com/temic/go-music/internal/metadata/store"
)

func TestStoreSaveAndGet(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "meta.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	record := model.AlbumRecord{
		AlbumID:       "album-1",
		Artist:        "Sabaton",
		Title:         "The Last Stand",
		Year:          2016,
		Genres:        []string{"Power Metal"},
		Description:   "Album description",
		MusicBrainzID: "mbid-1",
		CoverPath:     filepath.Join(t.TempDir(), "cover.jpg"),
		FetchStatus:   model.StatusSuccess,
		FetchedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.Save(ctx, record); err != nil {
		t.Fatal(err)
	}

	got, ok, err := s.Get(ctx, "album-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected record")
	}
	if got.Title != "The Last Stand" || got.Artist != "Sabaton" {
		t.Fatalf("unexpected record: %+v", got)
	}
	if len(got.Genres) != 1 || got.Genres[0] != "Power Metal" {
		t.Fatalf("unexpected genres: %+v", got.Genres)
	}
	if got.FetchStatus != model.StatusSuccess {
		t.Fatalf("status = %q", got.FetchStatus)
	}
}

func TestShouldFetchSkipsSuccessfulRecord(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "meta.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	if err := s.Save(ctx, model.AlbumRecord{
		AlbumID:     "album-1",
		Artist:      "Artist",
		Title:       "Album",
		Genres:      []string{},
		FetchStatus: model.StatusSuccess,
		FetchedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	shouldFetch, err := s.ShouldFetch(ctx, "album-1")
	if err != nil {
		t.Fatal(err)
	}
	if shouldFetch {
		t.Fatal("expected successful record to be skipped")
	}
}
