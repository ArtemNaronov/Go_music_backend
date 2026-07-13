package service

import (
	"fmt"
	"strings"
	"unicode"
)

func normalizeGenre(genre string) string {
	return strings.ToLower(strings.TrimSpace(genre))
}

func displayGenre(genre string) string {
	genre = strings.TrimSpace(genre)
	if genre == "" {
		return ""
	}

	runes := []rune(strings.ToLower(genre))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func stationSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '/':
			if b.Len() > 0 && b.String()[b.Len()-1] != '-' {
				b.WriteByte('-')
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func genreStationID(genre string) string {
	return "genre:" + stationSlug(normalizeGenre(genre))
}

func decadeStationID(decade int) string {
	return fmt.Sprintf("decade:%d", decade)
}

func decadeLabel(decade int) string {
	return fmt.Sprintf("%d-е", decade)
}

func trackDecade(year int) (int, bool) {
	if year < 1960 {
		return 0, false
	}
	return (year / 10) * 10, true
}
