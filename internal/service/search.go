package service

import (
	"context"
	"sort"
	"strings"

	"github.com/temic/go-music/internal/models"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 50
)

// SearchQuery configures unified library search.
type SearchQuery struct {
	Query string
	Limit int
}

func (s *LibraryService) Search(ctx context.Context, query SearchQuery) models.SearchResult {
	q := strings.ToLower(strings.TrimSpace(query.Query))
	limit := query.Limit
	if limit < 1 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	result := models.SearchResult{
		Query:   strings.TrimSpace(query.Query),
		Artists: []models.Artist{},
		Albums:  []models.Album{},
		Tracks:  []models.Track{},
	}
	if q == "" {
		return result
	}

	tracks := s.library.ListTracks(ctx)

	type artistAgg struct {
		name     string
		albums   map[string]struct{}
		tracks   int
		duration float64
		matched  bool
	}
	type albumAgg struct {
		title    string
		artist   string
		year     int
		tracks   int
		duration float64
		hasCover bool
		matched  bool
	}

	artists := make(map[string]*artistAgg)
	albums := make(map[string]*albumAgg)

	for _, track := range tracks {
		artistID := s.library.TrackArtistID(track)
		albumID := s.library.TrackAlbumID(track)
		folderArtist, folderAlbum := s.library.TrackFolderAlbum(track)
		trackMatch := trackFieldsMatch(track, q)

		aa := artists[artistID]
		if aa == nil {
			aa = &artistAgg{
				name:   folderArtist,
				albums: make(map[string]struct{}),
			}
			artists[artistID] = aa
		}
		aa.tracks++
		aa.duration += track.Duration
		aa.albums[albumID] = struct{}{}
		if strings.Contains(strings.ToLower(folderArtist), q) || trackMatch {
			aa.matched = true
		}

		ba := albums[albumID]
		if ba == nil {
			ba = &albumAgg{
				title:  folderAlbum,
				artist: folderArtist,
				year:   track.Year,
			}
			albums[albumID] = ba
		}
		ba.tracks++
		ba.duration += track.Duration
		if track.HasCover {
			ba.hasCover = true
		}
		if track.Year > ba.year {
			ba.year = track.Year
		}
		if strings.Contains(strings.ToLower(folderAlbum), q) ||
			strings.Contains(strings.ToLower(folderArtist), q) ||
			trackMatch {
			ba.matched = true
		}
	}

	matchedTracks := filterTracks(tracks, q)
	if len(matchedTracks) > limit {
		matchedTracks = matchedTracks[:limit]
	}
	result.Tracks = matchedTracks

	matchedArtists := make([]models.Artist, 0, limit)
	for artistID, aa := range artists {
		if !aa.matched {
			continue
		}
		matchedArtists = append(matchedArtists, models.Artist{
			ID:         artistID,
			Name:       aa.name,
			AlbumCount: len(aa.albums),
			TrackCount: aa.tracks,
			Duration:   aa.duration,
		})
	}
	sort.Slice(matchedArtists, func(i, j int) bool {
		return strings.ToLower(matchedArtists[i].Name) < strings.ToLower(matchedArtists[j].Name)
	})
	if len(matchedArtists) > limit {
		matchedArtists = matchedArtists[:limit]
	}
	result.Artists = matchedArtists

	matchedAlbums := make([]models.Album, 0, limit)
	for albumID, ba := range albums {
		if !ba.matched {
			continue
		}
		matchedAlbums = append(matchedAlbums, models.Album{
			ID:         albumID,
			Title:      ba.title,
			Artist:     ba.artist,
			Year:       ba.year,
			TrackCount: ba.tracks,
			Duration:   ba.duration,
			HasCover:   ba.hasCover,
		})
	}
	sort.Slice(matchedAlbums, func(i, j int) bool {
		ai := strings.ToLower(matchedAlbums[i].Artist + "\x00" + matchedAlbums[i].Title)
		aj := strings.ToLower(matchedAlbums[j].Artist + "\x00" + matchedAlbums[j].Title)
		return ai < aj
	})
	if len(matchedAlbums) > limit {
		matchedAlbums = matchedAlbums[:limit]
	}
	result.Albums = matchedAlbums

	return result
}

func trackFieldsMatch(track models.Track, query string) bool {
	return strings.Contains(strings.ToLower(track.Title), query) ||
		strings.Contains(strings.ToLower(track.Artist), query) ||
		strings.Contains(strings.ToLower(track.Album), query) ||
		strings.Contains(strings.ToLower(track.Genre), query)
}
