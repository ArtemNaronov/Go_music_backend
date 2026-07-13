package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/api"
	"github.com/temic/go-music/internal/library"
	"github.com/temic/go-music/internal/metadata"
	"github.com/temic/go-music/internal/metadata/model"
	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/internal/scanner"
	"github.com/temic/go-music/internal/service"
	"github.com/temic/go-music/pkg/id"
)

type stubProvider struct{}

func (stubProvider) Name() string { return "stub" }

func (stubProvider) SearchAlbum(context.Context, model.SearchQuery) (*model.Result, error) {
	return nil, nil
}

func TestListAlbumsByArtistID(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	queenPath := `D:\Music\Queen\Greatest Hits\01.mp3`
	store.Upsert(ctx, models.Track{
		ID: id.Track(queenPath), Path: queenPath,
		Title: "Song", Artist: "Queen", Album: "Greatest Hits",
	})

	metaSvc, err := metadata.NewService(metadata.Config{
		Enabled:    true,
		DataPath:   t.TempDir(),
		Provider:   stubProvider{},
		TrackAlbum: store.TrackAlbumID,
		FolderMeta: store.TrackFolderAlbum,
	}, zerolog.Nop())
	if err != nil {
		t.Fatal(err)
	}
	defer metaSvc.Close()

	sc := scanner.New(`D:\Music`, store, store, zerolog.Nop())
	svc := service.NewLibraryService(store, store, sc, metaSvc, zerolog.Nop())

	handler := api.NewHandler(svc, zerolog.Nop())
	router := api.NewRouter("secret", handler, zerolog.Nop())

	artistID := store.TrackArtistID(models.Track{Path: queenPath})
	req := httptest.NewRequest(http.MethodGet, "/api/albums?artist_id="+artistID, nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var page models.AlbumsPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode failed: %v, body = %s", err, rec.Body.String())
	}
	if page.Total != 1 {
		t.Fatalf("total = %d, want 1, body = %s", page.Total, rec.Body.String())
	}
}

func TestListAlbumsByArtistIDWithPendingMetadata(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	queenPath := `D:\Music\Queen\Greatest Hits\01.mp3`
	store.Upsert(ctx, models.Track{
		ID: id.Track(queenPath), Path: queenPath,
		Title: "Song", Artist: "Queen", Album: "Greatest Hits", Year: 1981,
	})

	dir := t.TempDir()
	metaSvc, err := metadata.NewService(metadata.Config{
		Enabled:    true,
		DataPath:   dir,
		Provider:   stubProvider{},
		TrackAlbum: store.TrackAlbumID,
		FolderMeta: store.TrackFolderAlbum,
	}, zerolog.Nop())
	if err != nil {
		t.Fatal(err)
	}
	defer metaSvc.Close()

	albumID := store.TrackAlbumID(models.Track{Path: queenPath})
	if err := metaSvc.ProcessJob(ctx, model.Job{
		AlbumID: albumID,
		Artist:  "Queen",
		Album:   "Greatest Hits",
		Year:    1981,
	}); err != nil {
		t.Fatal(err)
	}

	sc := scanner.New(`D:\Music`, store, store, zerolog.Nop())
	svc := service.NewLibraryService(store, store, sc, metaSvc, zerolog.Nop())
	handler := api.NewHandler(svc, zerolog.Nop())
	router := api.NewRouter("secret", handler, zerolog.Nop())

	artistID := store.TrackArtistID(models.Track{Path: queenPath})
	req := httptest.NewRequest(http.MethodGet, "/api/albums?artist_id="+artistID, nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	_ = filepath.Join(dir, "covers")
}
