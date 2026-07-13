package service

import (
	"context"

	"github.com/temic/go-music/internal/models"
)

type AlbumListQuery struct {
	Page     int
	Limit    int
	ArtistID string
}

func (s *LibraryService) ListArtistsPage(ctx context.Context, page, limit int) models.ArtistsPage {
	page, limit = normalizePagination(page, limit)
	artists := s.library.ListArtists(ctx)
	items, total, page, totalPages := paginate(artists, page, limit)

	return models.ArtistsPage{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}

func (s *LibraryService) ListAlbumsPage(ctx context.Context, query AlbumListQuery) models.AlbumsPage {
	page, limit := normalizePagination(query.Page, query.Limit)

	var albums []models.Album
	if query.ArtistID != "" {
		albums = s.library.ListAlbumsByArtist(ctx, query.ArtistID)
	} else {
		albums = s.library.ListAlbums(ctx)
	}

	items, total, page, totalPages := paginate(albums, page, limit)
	if hasEnricher(s.metadata) {
		items = s.metadata.EnrichAlbums(ctx, items)
	}

	return models.AlbumsPage{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}

func (s *LibraryService) ListTracks(ctx context.Context, query TrackListQuery) models.TracksPage {
	page, limit := normalizePagination(query.Page, query.Limit)
	query.Page = page
	query.Limit = limit

	tracks := applyTrackQuery(s.library, s.listTracksForQuery(ctx, query), query)
	items, total, page, totalPages := paginate(tracks, page, limit)

	return models.TracksPage{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}

func (s *LibraryService) listTracksForQuery(ctx context.Context, query TrackListQuery) []models.Track {
	switch {
	case query.AlbumID != "":
		return s.library.ListAlbumTracks(ctx, query.AlbumID)
	case query.ArtistID != "":
		return s.library.ListTracksByArtist(ctx, query.ArtistID)
	default:
		return s.library.ListTracks(ctx)
	}
}

// ResolveAlbumCover returns downloaded, embedded, or folder artwork for an album.
func (s *LibraryService) ResolveAlbumCover(ctx context.Context, albumID string) ([]byte, string, bool) {
	if hasEnricher(s.metadata) {
		if data, ok := s.metadata.AlbumCoverBytes(ctx, albumID); ok {
			return data, "image/jpeg", true
		}
	}

	tracks, ok := s.ListAlbumTracks(ctx, albumID)
	if !ok {
		return nil, "", false
	}

	for _, track := range tracks {
		if data, mime, ok := s.ResolveCover(ctx, track.ID); ok {
			return data, mime, true
		}
	}

	return nil, "", false
}
