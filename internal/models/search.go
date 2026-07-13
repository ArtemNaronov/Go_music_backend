package models

// SearchResult groups artists, albums, and tracks for a single query.
type SearchResult struct {
	Query   string  `json:"query"`
	Artists []Artist `json:"artists"`
	Albums  []Album  `json:"albums"`
	Tracks  []Track  `json:"tracks"`
}
