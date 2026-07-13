package library

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/internal/repository"
	"github.com/temic/go-music/pkg/id"
	"github.com/temic/go-music/pkg/musicpath"
)

type albumAgg struct {
	title    string
	artist   string
	year     int
	tracks   int
	duration float64
	hasCover bool
}

// Store is a thread-safe in-memory implementation of repository.Library.
type Store struct {
	mu           sync.RWMutex
	musicRoot    string
	tracks       map[string]models.Track
	paths        map[string]string
	albumTracks  map[string][]string
	albumMeta    map[string]*albumAgg
	artistTracks map[string][]string
	artistMeta   map[string]*artistMetaAgg
	artistAlbums map[string]map[string]struct{}
	covers       map[string]coverData
	cache        listCache
}

// NewStore creates an empty library store.
func NewStore(musicRoot string) *Store {
	return &Store{
		musicRoot:    filepath.Clean(musicRoot),
		tracks:       make(map[string]models.Track),
		paths:        make(map[string]string),
		albumTracks:  make(map[string][]string),
		albumMeta:    make(map[string]*albumAgg),
		artistTracks: make(map[string][]string),
		artistMeta:   make(map[string]*artistMetaAgg),
		artistAlbums: make(map[string]map[string]struct{}),
	}
}

var _ repository.Library = (*Store)(nil)

func (s *Store) GetTrack(_ context.Context, trackID string) (models.Track, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	track, ok := s.tracks[trackID]
	return track, ok
}

func (s *Store) GetTrackByPath(_ context.Context, path string) (models.Track, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	trackID, ok := s.paths[normalizePath(path)]
	if !ok {
		return models.Track{}, false
	}

	track, ok := s.tracks[trackID]
	return track, ok
}

func (s *Store) ListTracks(_ context.Context) []models.Track {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tracksByIDsLocked(s.allTrackIDsLocked())
}

func (s *Store) ListTracksByArtist(_ context.Context, artistID string) []models.Track {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tracksByIDsLocked(s.artistTracks[artistID])
}

func (s *Store) ListArtists(_ context.Context) []models.Artist {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cache.artistsValid {
		s.rebuildArtistCacheLocked()
	}

	return append([]models.Artist(nil), s.cache.artists...)
}

func (s *Store) ListAlbums(_ context.Context) []models.Album {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cache.albumsValid {
		s.rebuildAlbumCacheLocked()
	}

	return append([]models.Album(nil), s.cache.albums...)
}

func (s *Store) ListAlbumsByArtist(_ context.Context, artistID string) []models.Album {
	s.mu.RLock()
	defer s.mu.RUnlock()

	albumIDs, ok := s.artistAlbums[artistID]
	if !ok {
		return nil
	}

	return s.albumsByIDsLocked(albumIDs)
}

func (s *Store) GetAlbum(_ context.Context, albumID string) (models.Album, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agg, ok := s.albumMeta[albumID]
	if !ok {
		return models.Album{}, false
	}

	return albumFromAgg(albumID, agg), true
}

func (s *Store) ListAlbumTracks(_ context.Context, albumID string) []models.Track {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tracksByIDsLocked(s.albumTracks[albumID])
}

func (s *Store) aggregateAlbumLocked(albumID string) *albumAgg {
	trackIDs := s.albumTracks[albumID]
	if len(trackIDs) == 0 {
		return nil
	}

	var agg *albumAgg
	for _, trackID := range trackIDs {
		track, ok := s.tracks[trackID]
		if !ok {
			continue
		}

		if agg == nil {
			artist, title := musicpath.FolderArtistAlbum(s.musicRoot, track.Path)
			agg = &albumAgg{
				title:  title,
				artist: artist,
				year:   track.Year,
			}
		}

		agg.tracks++
		agg.duration += track.Duration
		if track.HasCover {
			agg.hasCover = true
		}
		if track.Year > agg.year {
			agg.year = track.Year
		}
	}

	return agg
}

func albumFromAgg(albumID string, agg *albumAgg) models.Album {
	return models.Album{
		ID:         albumID,
		Title:      agg.title,
		Artist:     agg.artist,
		Year:       agg.year,
		TrackCount: agg.tracks,
		Duration:   agg.duration,
		HasCover:   agg.hasCover,
	}
}

func (s *Store) albumKey(track models.Track) string {
	artist, album := musicpath.FolderArtistAlbum(s.musicRoot, track.Path)
	return id.Album(artist, album)
}

func (s *Store) TrackArtistID(track models.Track) string {
	artist, _ := musicpath.FolderArtistAlbum(s.musicRoot, track.Path)
	return id.Artist(artist)
}

func (s *Store) TrackAlbumID(track models.Track) string {
	return s.albumKey(track)
}

func (s *Store) TrackFolderAlbum(track models.Track) (string, string) {
	return musicpath.FolderArtistAlbum(s.musicRoot, track.Path)
}

func (s *Store) Upsert(_ context.Context, track models.Track) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedPath := normalizePath(track.Path)

	if existingID, ok := s.paths[normalizedPath]; ok && existingID != track.ID {
		if existing, found := s.tracks[existingID]; found {
			s.unindexTrackLocked(existing)
		}
		delete(s.tracks, existingID)
		delete(s.covers, existingID)
	}

	if previous, ok := s.tracks[track.ID]; ok {
		delete(s.paths, normalizePath(previous.Path))
		s.unindexTrackLocked(previous)
		if track.ID != previous.ID {
			delete(s.covers, previous.ID)
		}
	}

	s.tracks[track.ID] = track
	s.paths[normalizedPath] = track.ID
	s.indexTrackLocked(track)
	s.invalidateListCacheLocked()
}

func (s *Store) Remove(_ context.Context, trackID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	track, ok := s.tracks[trackID]
	if !ok {
		return false
	}

	s.unindexTrackLocked(track)
	delete(s.tracks, trackID)
	delete(s.paths, normalizePath(track.Path))
	delete(s.covers, trackID)
	s.invalidateListCacheLocked()
	return true
}

func (s *Store) RemoveByPath(_ context.Context, path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedPath := normalizePath(path)
	trackID, ok := s.paths[normalizedPath]
	if !ok {
		return false
	}

	track := s.tracks[trackID]
	s.unindexTrackLocked(track)
	delete(s.tracks, trackID)
	delete(s.paths, normalizedPath)
	delete(s.covers, trackID)
	s.invalidateListCacheLocked()
	return true
}

func (s *Store) ReplaceAll(_ context.Context, tracks []models.Track) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tracks = make(map[string]models.Track, len(tracks))
	s.paths = make(map[string]string, len(tracks))
	s.clearCoversLocked()

	for _, track := range tracks {
		s.tracks[track.ID] = track
		s.paths[normalizePath(track.Path)] = track.ID
	}

	s.rebuildIndicesLocked()
	s.invalidateListCacheLocked()
}

func (s *Store) Count(_ context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tracks)
}

func (s *Store) allTrackIDsLocked() []string {
	ids := make([]string, 0, len(s.tracks))
	for trackID := range s.tracks {
		ids = append(ids, trackID)
	}
	return ids
}

func normalizePath(path string) string {
	return strings.ToLower(filepath.Clean(path))
}

func sortTracks(tracks []models.Track) {
	sort.Slice(tracks, func(i, j int) bool {
		a, b := tracks[i], tracks[j]

		artistCmp := strings.ToLower(a.Artist)
		artistCmpB := strings.ToLower(b.Artist)
		if artistCmp != artistCmpB {
			return artistCmp < artistCmpB
		}

		albumCmp := strings.ToLower(a.Album)
		albumCmpB := strings.ToLower(b.Album)
		if albumCmp != albumCmpB {
			return albumCmp < albumCmpB
		}

		if a.TrackNumber != b.TrackNumber {
			return a.TrackNumber < b.TrackNumber
		}

		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	})
}
