package library

import (
	"sort"
	"strings"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/pkg/id"
	"github.com/temic/go-music/pkg/musicpath"
)

type artistMetaAgg struct {
	name     string
	albumIDs map[string]struct{}
	tracks   int
	duration float64
}

type listCache struct {
	artists      []models.Artist
	albums       []models.Album
	artistsValid bool
	albumsValid  bool
}

func (s *Store) trackArtistIDLocked(track models.Track) string {
	artist, _ := musicpath.FolderArtistAlbum(s.musicRoot, track.Path)
	return id.Artist(artist)
}

func (s *Store) invalidateListCacheLocked() {
	s.cache.artistsValid = false
	s.cache.albumsValid = false
}

func (s *Store) indexTrackLocked(track models.Track) {
	artistID := s.trackArtistIDLocked(track)
	albumID := s.albumKey(track)

	s.albumTracks[albumID] = append(s.albumTracks[albumID], track.ID)
	s.artistTracks[artistID] = append(s.artistTracks[artistID], track.ID)

	if s.artistAlbums[artistID] == nil {
		s.artistAlbums[artistID] = make(map[string]struct{})
	}
	s.artistAlbums[artistID][albumID] = struct{}{}

	s.addAlbumMetaLocked(albumID, track)
	s.addArtistMetaLocked(artistID, albumID, track)
}

func (s *Store) unindexTrackLocked(track models.Track) {
	artistID := s.trackArtistIDLocked(track)
	albumID := s.albumKey(track)

	s.removeTrackIDLocked(s.albumTracks, albumID, track.ID)
	s.removeTrackIDLocked(s.artistTracks, artistID, track.ID)

	if len(s.albumTracks[albumID]) == 0 {
		delete(s.albumTracks, albumID)
		delete(s.albumMeta, albumID)
	} else {
		s.refreshAlbumMetaLocked(albumID)
	}

	if len(s.artistTracks[artistID]) == 0 {
		delete(s.artistTracks, artistID)
		delete(s.artistMeta, artistID)
		delete(s.artistAlbums, artistID)
	} else {
		s.refreshArtistMetaLocked(artistID)
	}
}

func (s *Store) removeTrackIDLocked(index map[string][]string, key, trackID string) {
	ids := index[key]
	for i, id := range ids {
		if id != trackID {
			continue
		}
		index[key] = append(ids[:i], ids[i+1:]...)
		return
	}
}

func (s *Store) addAlbumMetaLocked(albumID string, track models.Track) {
	agg := s.albumMeta[albumID]
	if agg == nil {
		artist, title := musicpath.FolderArtistAlbum(s.musicRoot, track.Path)
		agg = &albumAgg{
			title:  title,
			artist: artist,
			year:   track.Year,
		}
		s.albumMeta[albumID] = agg
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

func (s *Store) addArtistMetaLocked(artistID, albumID string, track models.Track) {
	agg := s.artistMeta[artistID]
	if agg == nil {
		artistName, _ := musicpath.FolderArtistAlbum(s.musicRoot, track.Path)
		agg = &artistMetaAgg{
			name:     artistName,
			albumIDs: make(map[string]struct{}),
		}
		s.artistMeta[artistID] = agg
	}

	agg.tracks++
	agg.duration += track.Duration
	agg.albumIDs[albumID] = struct{}{}
}

func (s *Store) refreshAlbumMetaLocked(albumID string) {
	if agg := s.aggregateAlbumLocked(albumID); agg == nil {
		delete(s.albumMeta, albumID)
		return
	} else {
		s.albumMeta[albumID] = agg
	}
}

func (s *Store) refreshArtistMetaLocked(artistID string) {
	trackIDs := s.artistTracks[artistID]
	if len(trackIDs) == 0 {
		delete(s.artistMeta, artistID)
		delete(s.artistAlbums, artistID)
		return
	}

	agg := &artistMetaAgg{
		albumIDs: make(map[string]struct{}),
	}

	for _, trackID := range trackIDs {
		track, ok := s.tracks[trackID]
		if !ok {
			continue
		}

		if agg.name == "" {
			artistName, _ := musicpath.FolderArtistAlbum(s.musicRoot, track.Path)
			agg.name = artistName
		}

		agg.tracks++
		agg.duration += track.Duration
		agg.albumIDs[s.albumKey(track)] = struct{}{}
	}

	if agg.tracks == 0 {
		delete(s.artistMeta, artistID)
		delete(s.artistAlbums, artistID)
		return
	}

	s.artistMeta[artistID] = agg
	s.artistAlbums[artistID] = mapsCopy(agg.albumIDs)
}

func mapsCopy(src map[string]struct{}) map[string]struct{} {
	dst := make(map[string]struct{}, len(src))
	for key := range src {
		dst[key] = struct{}{}
	}
	return dst
}

func (s *Store) rebuildIndicesLocked() {
	s.albumTracks = make(map[string][]string, len(s.tracks)/10+1)
	s.albumMeta = make(map[string]*albumAgg, len(s.tracks)/10+1)
	s.artistTracks = make(map[string][]string, len(s.tracks)/20+1)
	s.artistMeta = make(map[string]*artistMetaAgg, len(s.tracks)/20+1)
	s.artistAlbums = make(map[string]map[string]struct{}, len(s.tracks)/20+1)

	for _, track := range s.tracks {
		s.indexTrackLocked(track)
	}
}

func (s *Store) rebuildArtistCacheLocked() {
	result := make([]models.Artist, 0, len(s.artistMeta))
	for artistID, agg := range s.artistMeta {
		result = append(result, models.Artist{
			ID:         artistID,
			Name:       agg.name,
			AlbumCount: len(agg.albumIDs),
			TrackCount: agg.tracks,
			Duration:   agg.duration,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	s.cache.artists = result
	s.cache.artistsValid = true
}

func (s *Store) rebuildAlbumCacheLocked() {
	result := make([]models.Album, 0, len(s.albumMeta))
	for albumID, agg := range s.albumMeta {
		result = append(result, albumFromAgg(albumID, agg))
	}

	sort.Slice(result, func(i, j int) bool {
		ai := strings.ToLower(result[i].Artist + "\x00" + result[i].Title)
		aj := strings.ToLower(result[j].Artist + "\x00" + result[j].Title)
		return ai < aj
	})

	s.cache.albums = result
	s.cache.albumsValid = true
}

func (s *Store) tracksByIDsLocked(trackIDs []string) []models.Track {
	tracks := make([]models.Track, 0, len(trackIDs))
	for _, trackID := range trackIDs {
		if track, ok := s.tracks[trackID]; ok {
			tracks = append(tracks, track)
		}
	}
	sortTracks(tracks)
	return tracks
}

func (s *Store) albumsByIDsLocked(albumIDs map[string]struct{}) []models.Album {
	result := make([]models.Album, 0, len(albumIDs))
	for albumID := range albumIDs {
		if agg, ok := s.albumMeta[albumID]; ok {
			result = append(result, albumFromAgg(albumID, agg))
		}
	}

	sort.Slice(result, func(i, j int) bool {
		ai := strings.ToLower(result[i].Artist + "\x00" + result[i].Title)
		aj := strings.ToLower(result[j].Artist + "\x00" + result[j].Title)
		return ai < aj
	})

	return result
}
