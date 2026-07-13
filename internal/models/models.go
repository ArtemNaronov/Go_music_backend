package models

import "time"

// Track represents a single audio file in the library.
type Track struct {
	ID          string    `json:"id"`
	Path        string    `json:"-"`
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	Album       string    `json:"album"`
	Year        int       `json:"year,omitempty"`
	Genre       string    `json:"genre,omitempty"`
	Duration    float64   `json:"duration"`
	TrackNumber int       `json:"track_number,omitempty"`
	DiscNumber  int       `json:"disc_number,omitempty"`
	HasCover    bool      `json:"has_cover"`
	Size        int64     `json:"size"`
	Format      string    `json:"format"`
	ModifiedAt  time.Time `json:"modified_at"`
}

// Album groups tracks by artist + album name.
type Album struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Artist         string   `json:"artist"`
	Year           int      `json:"year,omitempty"`
	TrackCount     int      `json:"track_count"`
	Duration       float64  `json:"duration"`
	HasCover       bool     `json:"has_cover"`
	CoverURL       string   `json:"cover_url,omitempty"`
	Genres         []string `json:"genres,omitempty"`
	Description    string   `json:"description,omitempty"`
	MusicBrainzID  string   `json:"musicbrainz_id,omitempty"`
	MetadataStatus string   `json:"metadata_status,omitempty"`
}

// Artist groups albums and tracks by artist name.
type Artist struct {
	ID         string  `json:"artist"`
	Name       string  `json:"name"`
	AlbumCount int     `json:"album_count"`
	TrackCount int     `json:"track_count"`
	Duration   float64 `json:"duration"`
}

// ScanResult holds statistics from a library scan.
type ScanResult struct {
	TracksFound int     `json:"tracks_found"`
	Duration    float64 `json:"duration"` // seconds
}

// ArtistsPage is a paginated list of artists.
type ArtistsPage struct {
	Items      []Artist `json:"items"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	TotalPages int      `json:"total_pages"`
}

// AlbumsPage is a paginated list of albums.
type AlbumsPage struct {
	Items      []Album `json:"items"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalPages int     `json:"total_pages"`
}

// TracksPage is a paginated list of tracks.
type TracksPage struct {
	Items      []Track `json:"items"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalPages int     `json:"total_pages"`
}

// APIError is the standard error response envelope.
type APIError struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}
