package service

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/temic/go-music/internal/models"
)

func (s *LibraryService) ListStations(ctx context.Context) []models.Station {
	tracks := s.library.ListTracks(ctx)
	if len(tracks) == 0 {
		return []models.Station{}
	}

	stations := []models.Station{
		{
			ID:          "all",
			Name:        "Всё радио",
			Kind:        "all",
			Description: "Случайные треки из всей библиотеки",
			TrackCount:  len(tracks),
		},
	}

	type genreAgg struct {
		display string
		count   int
	}
	genres := make(map[string]*genreAgg)
	decades := make(map[int]int)

	for _, track := range tracks {
		if genre := normalizeGenre(track.Genre); genre != "" {
			agg, ok := genres[genre]
			if !ok {
				agg = &genreAgg{display: displayGenre(track.Genre)}
				genres[genre] = agg
			}
			agg.count++
		}

		if decade, ok := trackDecade(track.Year); ok {
			decades[decade]++
		}
	}

	genreStations := make([]models.Station, 0, len(genres))
	for genre, agg := range genres {
		genreStations = append(genreStations, models.Station{
			ID:          genreStationID(genre),
			Name:        agg.display,
			Kind:        "genre",
			Description: "Треки жанра " + agg.display,
			TrackCount:  agg.count,
		})
	}
	sort.Slice(genreStations, func(i, j int) bool {
		if genreStations[i].TrackCount != genreStations[j].TrackCount {
			return genreStations[i].TrackCount > genreStations[j].TrackCount
		}
		return strings.ToLower(genreStations[i].Name) < strings.ToLower(genreStations[j].Name)
	})
	stations = append(stations, genreStations...)

	decadeStations := make([]models.Station, 0, len(decades))
	for decade, count := range decades {
		decadeStations = append(decadeStations, models.Station{
			ID:          decadeStationID(decade),
			Name:        decadeLabel(decade),
			Kind:        "decade",
			Description: "Музыка " + decadeLabel(decade),
			TrackCount:  count,
		})
	}
	sort.Slice(decadeStations, func(i, j int) bool {
		return decadeStations[i].Name > decadeStations[j].Name
	})
	stations = append(stations, decadeStations...)

	return stations
}

func trackMatchesStation(track models.Track, stationID string) bool {
	stationID = strings.TrimSpace(stationID)
	if stationID == "" || stationID == "all" {
		return true
	}

	kind, value, ok := strings.Cut(stationID, ":")
	if !ok || value == "" {
		return false
	}

	switch kind {
	case "genre":
		return stationSlug(normalizeGenre(track.Genre)) == value
	case "decade":
		decade, err := strconv.Atoi(value)
		if err != nil {
			return false
		}
		trackDecade, ok := trackDecade(track.Year)
		return ok && trackDecade == decade
	default:
		return false
	}
}

func trackMatchesSeed(library repositoryLibrary, track, seed models.Track) bool {
	if track.ID == seed.ID {
		return true
	}
	if library.TrackArtistID(track) == library.TrackArtistID(seed) {
		return true
	}

	seedGenre := normalizeGenre(seed.Genre)
	if seedGenre != "" && normalizeGenre(track.Genre) == seedGenre {
		return true
	}

	seedDecade, seedOK := trackDecade(seed.Year)
	trackDecadeValue, trackOK := trackDecade(track.Year)
	return seedOK && trackOK && seedDecade == trackDecadeValue
}

// repositoryLibrary is the subset of repository.Library used by radio/search helpers.
type repositoryLibrary interface {
	TrackArtistID(track models.Track) string
	TrackAlbumID(track models.Track) string
}
