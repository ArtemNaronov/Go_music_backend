package service

import (
	"sort"
	"strings"

	"github.com/temic/go-music/internal/models"
	"github.com/temic/go-music/internal/repository"
)

// TrackListQuery configures track listing.
type TrackListQuery struct {
	Page     int
	Limit    int
	Search   string
	ArtistID string
	AlbumID  string
	Sort     string
}

func applyTrackQuery(library repository.Library, tracks []models.Track, query TrackListQuery) []models.Track {
	filtered := make([]models.Track, 0, len(tracks))
	for _, track := range tracks {
		if query.ArtistID != "" && library.TrackArtistID(track) != query.ArtistID {
			continue
		}
		if query.AlbumID != "" && library.TrackAlbumID(track) != query.AlbumID {
			continue
		}
		filtered = append(filtered, track)
	}

	filtered = filterTracks(filtered, query.Search)
	sortTracks(filtered, query.Sort)
	return filtered
}

func sortTracks(tracks []models.Track, sortParam string) {
	field, desc := parseSort(sortParam)
	if field == "" {
		return
	}

	sort.SliceStable(tracks, func(i, j int) bool {
		less := compareTracks(tracks[i], tracks[j], field)
		if desc {
			return !less
		}
		return less
	})
}

func parseSort(sortParam string) (field string, desc bool) {
	sortParam = strings.TrimSpace(sortParam)
	if sortParam == "" {
		return "", false
	}

	if strings.HasPrefix(sortParam, "-") {
		return strings.TrimPrefix(sortParam, "-"), true
	}
	return sortParam, false
}

func compareTracks(a, b models.Track, field string) bool {
	switch field {
	case "title":
		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	case "artist":
		return strings.ToLower(a.Artist) < strings.ToLower(b.Artist)
	case "album":
		return strings.ToLower(a.Album) < strings.ToLower(b.Album)
	case "track_number":
		if a.TrackNumber != b.TrackNumber {
			return a.TrackNumber < b.TrackNumber
		}
		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	case "duration":
		return a.Duration < b.Duration
	case "modified_at":
		return a.ModifiedAt.Before(b.ModifiedAt)
	default:
		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	}
}

func paginate[T any](items []T, page, limit int) ([]T, int, int, int) {
	page, limit = normalizePagination(page, limit)
	total := len(items)
	totalPages := calcTotalPages(total, limit)

	start := (page - 1) * limit
	if start >= total {
		return []T{}, total, page, totalPages
	}

	end := start + limit
	if end > total {
		end = total
	}

	return append([]T(nil), items[start:end]...), total, page, totalPages
}
