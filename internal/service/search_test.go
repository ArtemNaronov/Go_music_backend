package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/library"
	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
)

func TestSearchReturnsArtistsAlbumsAndTracks(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\Kings of Leon\Only by the Night\Sex on Fire.mp3`),
		Path: `D:\Music\Kings of Leon\Only by the Night\Sex on Fire.mp3`,
		Title: "Sex on Fire", Artist: "Kings of Leon", Album: "Only by the Night", Genre: "Rock",
	})
	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\Queen\Greatest Hits\Bohemian Rhapsody.mp3`),
		Path: `D:\Music\Queen\Greatest Hits\Bohemian Rhapsody.mp3`,
		Title: "Bohemian Rhapsody", Artist: "Queen", Album: "Greatest Hits", Genre: "Rock",
	})

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())
	result := svc.Search(ctx, SearchQuery{Query: "bohemian", Limit: 10})

	if len(result.Tracks) != 1 || result.Tracks[0].Title != "Bohemian Rhapsody" {
		t.Fatalf("unexpected tracks: %+v", result.Tracks)
	}
	if len(result.Artists) != 1 || result.Artists[0].Name != "Queen" {
		t.Fatalf("unexpected artists: %+v", result.Artists)
	}
	if len(result.Albums) != 1 || result.Albums[0].Title != "Greatest Hits" {
		t.Fatalf("unexpected albums: %+v", result.Albums)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())

	result := svc.Search(context.Background(), SearchQuery{Query: "   "})
	if len(result.Artists) != 0 || len(result.Albums) != 0 || len(result.Tracks) != 0 {
		t.Fatalf("expected empty result, got %+v", result)
	}
}

func TestListStationsIncludesAllGenreAndDecade(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\Artist\Album\01.mp3`),
		Path: `D:\Music\Artist\Album\01.mp3`,
		Title: "One", Artist: "Artist", Album: "Album", Genre: "Alternative Rock", Year: 2008,
	})
	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\Artist\Album\02.mp3`),
		Path: `D:\Music\Artist\Album\02.mp3`,
		Title: "Two", Artist: "Artist", Album: "Album", Genre: "Rock", Year: 2012,
	})

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())
	stations := svc.ListStations(ctx)

	if len(stations) < 4 {
		t.Fatalf("expected at least 4 stations, got %d: %+v", len(stations), stations)
	}
	if stations[0].ID != "all" {
		t.Fatalf("first station = %q, want all", stations[0].ID)
	}

	ids := make(map[string]struct{})
	for _, station := range stations {
		ids[station.ID] = struct{}{}
	}
	if _, ok := ids["genre:alternative-rock"]; !ok {
		t.Fatalf("missing genre station, got %+v", stations)
	}
	if _, ok := ids["genre:rock"]; !ok {
		t.Fatalf("missing rock station, got %+v", stations)
	}
	if _, ok := ids["decade:2000"]; !ok {
		t.Fatalf("missing 2000s station, got %+v", stations)
	}
}

func TestRadioByArtistID(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	queen := models.Track{
		ID: id.Track(`D:\Music\Queen\Album\01.mp3`), Path: `D:\Music\Queen\Album\01.mp3`,
		Title: "Queen Track", Artist: "Queen", Album: "Album",
	}
	other := models.Track{
		ID: id.Track(`D:\Music\Artist\Album\01.mp3`), Path: `D:\Music\Artist\Album\01.mp3`,
		Title: "Other Track", Artist: "Artist", Album: "Album",
	}
	store.Upsert(ctx, queen)
	store.Upsert(ctx, other)

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())
	artistID := store.TrackArtistID(queen)

	queue, ok := svc.Radio(ctx, RadioQuery{Limit: 10, ArtistID: artistID})
	if !ok {
		t.Fatal("expected artist radio")
	}
	if len(queue.Items) != 1 || queue.Items[0].ID != queen.ID {
		t.Fatalf("unexpected queue: %+v", queue.Items)
	}
}

func TestRadioByStation(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\Artist\Album\01.mp3`), Path: `D:\Music\Artist\Album\01.mp3`,
		Title: "Rock Track", Artist: "Artist", Album: "Album", Genre: "Rock",
	})
	store.Upsert(ctx, models.Track{
		ID: id.Track(`D:\Music\Artist\Album\02.mp3`), Path: `D:\Music\Artist\Album\02.mp3`,
		Title: "Pop Track", Artist: "Artist", Album: "Album", Genre: "Pop",
	})

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())
	queue, ok := svc.Radio(ctx, RadioQuery{Limit: 10, Station: "genre:rock"})
	if !ok {
		t.Fatal("expected station radio")
	}
	if len(queue.Items) != 1 || queue.Items[0].Title != "Rock Track" {
		t.Fatalf("unexpected queue: %+v", queue.Items)
	}
}

func TestRadioBySeed(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	seed := models.Track{
		ID: id.Track(`D:\Music\Queen\Album\01.mp3`), Path: `D:\Music\Queen\Album\01.mp3`,
		Title: "Seed", Artist: "Queen", Album: "Album", Genre: "Rock", Year: 1975,
	}
	sameArtist := models.Track{
		ID: id.Track(`D:\Music\Queen\Album\02.mp3`), Path: `D:\Music\Queen\Album\02.mp3`,
		Title: "Same Artist", Artist: "Queen", Album: "Album", Genre: "Pop", Year: 1980,
	}
	sameGenre := models.Track{
		ID: id.Track(`D:\Music\Artist\Album\01.mp3`), Path: `D:\Music\Artist\Album\01.mp3`,
		Title: "Same Genre", Artist: "Artist", Album: "Album", Genre: "Rock", Year: 2000,
	}
	other := models.Track{
		ID: id.Track(`D:\Music\Artist\Album\02.mp3`), Path: `D:\Music\Artist\Album\02.mp3`,
		Title: "Other", Artist: "Artist", Album: "Album", Genre: "Jazz", Year: 1950,
	}

	for _, track := range []models.Track{seed, sameArtist, sameGenre, other} {
		store.Upsert(ctx, track)
	}

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())
	queue, ok := svc.Radio(ctx, RadioQuery{Limit: 10, Seed: seed.ID})
	if !ok {
		t.Fatal("expected seed radio")
	}
	if queue.TotalAvailable != 3 {
		t.Fatalf("total available = %d, want 3", queue.TotalAvailable)
	}
}

func TestRadioInvalidSeed(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())

	if _, ok := svc.Radio(context.Background(), RadioQuery{Seed: "missing"}); ok {
		t.Fatal("expected invalid seed to fail")
	}
}

func TestRadioReturnsShuffledTracks(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		path := fmt.Sprintf(`D:\Music\Artist\Album\%02d.mp3`, i)
		store.Upsert(ctx, models.Track{
			ID: id.Track(path), Path: path,
			Title: "Track", Artist: "Artist", Album: "Album",
		})
	}

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())

	queue, ok := svc.Radio(ctx, RadioQuery{Limit: 3})
	if !ok {
		t.Fatal("expected radio queue")
	}
	if queue.Returned != 3 || queue.TotalAvailable != 5 || len(queue.Items) != 3 {
		t.Fatalf("unexpected queue: %+v", queue)
	}

	seen := make(map[string]struct{})
	for _, track := range queue.Items {
		if _, exists := seen[track.ID]; exists {
			t.Fatalf("duplicate track in queue: %s", track.ID)
		}
		seen[track.ID] = struct{}{}
	}
}

func TestRadioExclude(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	ctx := context.Background()

	first := models.Track{
		ID: id.Track(`D:\Music\Artist\Album\01.mp3`), Path: `D:\Music\Artist\Album\01.mp3`,
		Title: "One", Artist: "Artist", Album: "Album",
	}
	second := models.Track{
		ID: id.Track(`D:\Music\Artist\Album\02.mp3`), Path: `D:\Music\Artist\Album\02.mp3`,
		Title: "Two", Artist: "Artist", Album: "Album",
	}
	store.Upsert(ctx, first)
	store.Upsert(ctx, second)

	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())

	queue, ok := svc.Radio(ctx, RadioQuery{Limit: 5, Exclude: []string{first.ID}})
	if !ok {
		t.Fatal("expected radio queue")
	}
	if len(queue.Items) != 1 || queue.Items[0].ID != second.ID {
		t.Fatalf("unexpected queue: %+v", queue.Items)
	}
}

func TestRadioEmptyLibrary(t *testing.T) {
	store := library.NewStore(`D:\Music`)
	svc := NewLibraryService(store, store, stubIndexer{}, nil, zerolog.Nop())

	if _, ok := svc.Radio(context.Background(), RadioQuery{}); ok {
		t.Fatal("expected empty radio to fail")
	}
}
