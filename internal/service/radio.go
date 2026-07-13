package service

import (
	"context"
	"crypto/rand"
	"math/big"

	"github.com/temic/go-music/internal/models"
)

const (
	defaultRadioLimit = 20
	maxRadioLimit     = 50
)

// RadioQuery configures random radio track selection.
type RadioQuery struct {
	Limit    int
	Exclude  []string
	ArtistID string
	Station  string
	Seed     string
}

func (s *LibraryService) Radio(ctx context.Context, query RadioQuery) (models.RadioQueue, bool) {
	limit := query.Limit
	if limit < 1 {
		limit = defaultRadioLimit
	}
	if limit > maxRadioLimit {
		limit = maxRadioLimit
	}

	exclude := toExcludeSet(query.Exclude)
	var seedTrack models.Track
	if query.Seed != "" {
		track, ok := s.library.GetTrack(ctx, query.Seed)
		if !ok {
			return models.RadioQueue{}, false
		}
		seedTrack = track
	}

	candidates := make([]models.Track, 0)
	for _, track := range s.library.ListTracks(ctx) {
		if _, skip := exclude[track.ID]; skip {
			continue
		}
		if query.ArtistID != "" && s.library.TrackArtistID(track) != query.ArtistID {
			continue
		}
		if query.Station != "" && !trackMatchesStation(track, query.Station) {
			continue
		}
		if query.Seed != "" && !trackMatchesSeed(s.library, track, seedTrack) {
			continue
		}
		candidates = append(candidates, track)
	}

	total := len(candidates)
	if total == 0 {
		return models.RadioQueue{}, false
	}

	shuffleTracks(candidates)

	if limit > total {
		limit = total
	}

	return models.RadioQueue{
		Items:          append([]models.Track(nil), candidates[:limit]...),
		TotalAvailable: total,
		Returned:       limit,
	}, true
}

func toExcludeSet(ids []string) map[string]struct{} {
	if len(ids) == 0 {
		return map[string]struct{}{}
	}

	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		set[id] = struct{}{}
	}
	return set
}

func shuffleTracks(tracks []models.Track) {
	for i := len(tracks) - 1; i > 0; i-- {
		j := randomInt(i + 1)
		tracks[i], tracks[j] = tracks[j], tracks[i]
	}
}

func randomInt(max int) int {
	if max <= 1 {
		return 0
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}
