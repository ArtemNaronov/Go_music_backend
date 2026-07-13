package service

import (
	"context"
	"reflect"

	"github.com/temic/go-music/internal/models"
)

// AlbumEnricher merges external metadata into API album responses.
type AlbumEnricher interface {
	EnrichAlbum(ctx context.Context, album models.Album) models.Album
	EnrichAlbums(ctx context.Context, albums []models.Album) []models.Album
	AlbumCoverBytes(ctx context.Context, albumID string) ([]byte, bool)
}

func normalizeEnricher(enricher AlbumEnricher) AlbumEnricher {
	if !hasEnricher(enricher) {
		return nil
	}
	return enricher
}

func hasEnricher(enricher AlbumEnricher) bool {
	if enricher == nil {
		return false
	}
	value := reflect.ValueOf(enricher)
	return value.Kind() != reflect.Pointer || !value.IsNil()
}
