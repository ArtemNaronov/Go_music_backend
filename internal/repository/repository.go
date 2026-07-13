package repository

import (
	"context"

	"github.com/temic/go-music/internal/models"
)

// Library provides access to the in-memory music library.
// A future SQLite implementation can satisfy the same interface.
type Library interface {
	GetTrack(ctx context.Context, id string) (models.Track, bool)
	GetTrackByPath(ctx context.Context, path string) (models.Track, bool)
	ListTracks(ctx context.Context) []models.Track
	ListTracksByArtist(ctx context.Context, artistID string) []models.Track
	ListArtists(ctx context.Context) []models.Artist
	ListAlbums(ctx context.Context) []models.Album
	ListAlbumsByArtist(ctx context.Context, artistID string) []models.Album
	GetAlbum(ctx context.Context, albumID string) (models.Album, bool)
	ListAlbumTracks(ctx context.Context, albumID string) []models.Track
	Upsert(ctx context.Context, track models.Track)
	Remove(ctx context.Context, id string) bool
	RemoveByPath(ctx context.Context, path string) bool
	ReplaceAll(ctx context.Context, tracks []models.Track)
	Count(ctx context.Context) int
	TrackArtistID(track models.Track) string
	TrackAlbumID(track models.Track) string
	TrackFolderAlbum(track models.Track) (artist, album string)
}
