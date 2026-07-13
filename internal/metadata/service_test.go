package metadata_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/metadata"
	"github.com/temic/go-music/internal/metadata/model"
	"github.com/temic/go-music/internal/models"
)

type stubProvider struct {
	result *model.Result
	err    error
	calls  int
}

func (p *stubProvider) Name() string { return "stub" }

func (p *stubProvider) SearchAlbum(context.Context, model.SearchQuery) (*model.Result, error) {
	p.calls++
	return p.result, p.err
}

func TestServiceEnrichAlbum(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc, err := metadata.NewService(metadata.Config{
		Enabled:    true,
		DataPath:   dir,
		Provider:   &stubProvider{},
		TrackAlbum: func(track models.Track) string { return "album-1" },
		FolderMeta: func(track models.Track) (string, string) { return "Sabaton", "The Last Stand" },
	}, zerolog.Nop())
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	ctx := context.Background()
	svc.OnTrackIndexed(ctx, models.Track{
		ID: "track-1", Path: `D:\Music\Sabaton\The Last Stand\01.mp3`,
		Title: "Sparta", Artist: "Sabaton", Album: "The Last Stand", Year: 2016,
	})

	album := svc.EnrichAlbum(ctx, models.Album{
		ID: "album-1", Title: "The Last Stand", Artist: "Sabaton", HasCover: false,
	})
	if album.MetadataStatus != string(model.StatusPending) {
		t.Fatalf("status = %q, want pending", album.MetadataStatus)
	}
}

func TestServiceProcessJobStoresMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	provider := &stubProvider{
		result: &model.Result{
			MusicBrainzID: "mbid-1",
			Title:         "The Last Stand",
			Artist:        "Sabaton",
			Year:          2016,
			Genres:        []string{"Power Metal"},
			Description:   "Swedish metal",
		},
	}

	svc, err := metadata.NewService(metadata.Config{
		Enabled:    true,
		DataPath:   dir,
		Provider:   provider,
		TrackAlbum: func(track models.Track) string { return "album-1" },
		FolderMeta: func(track models.Track) (string, string) { return "Sabaton", "The Last Stand" },
	}, zerolog.Nop())
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	ctx := context.Background()
	if err := svc.ProcessJob(ctx, model.Job{
		AlbumID: "album-1",
		Artist:  "Sabaton",
		Album:   "The Last Stand",
		Year:    2016,
	}); err != nil {
		t.Fatal(err)
	}

	album := svc.EnrichAlbum(ctx, models.Album{
		ID: "album-1", Title: "The Last Stand", Artist: "Sabaton",
	})
	if album.MusicBrainzID != "mbid-1" {
		t.Fatalf("mbid = %q", album.MusicBrainzID)
	}
	if album.Description != "Swedish metal" {
		t.Fatalf("description = %q", album.Description)
	}
	if len(album.Genres) != 1 || album.Genres[0] != "Power Metal" {
		t.Fatalf("genres = %+v", album.Genres)
	}
	if album.MetadataStatus != string(model.StatusSuccess) {
		t.Fatalf("status = %q", album.MetadataStatus)
	}
}
