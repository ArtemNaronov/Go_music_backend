package model

import (
	"time"
)

// FetchStatus describes metadata enrichment state for an album.
type FetchStatus string

const (
	StatusPending FetchStatus = "pending"
	StatusSuccess FetchStatus = "success"
	StatusFailed  FetchStatus = "failed"
	StatusSkipped FetchStatus = "skipped"
)

// SearchQuery identifies an album for external metadata lookup.
type SearchQuery struct {
	AlbumID string
	Artist  string
	Album   string
	Year    int
}

// Result is enriched album metadata from a provider.
type Result struct {
	MusicBrainzID string
	Title         string
	Artist        string
	Year          int
	Genres        []string
	Description   string
	CoverImageURL string
}

// AlbumRecord is persisted enrichment data for an album.
type AlbumRecord struct {
	AlbumID       string
	Artist        string
	Title         string
	Year          int
	Genres        []string
	Description   string
	MusicBrainzID string
	CoverPath     string
	FetchStatus   FetchStatus
	FetchedAt     time.Time
	UpdatedAt     time.Time
}

// Job is a background metadata fetch task.
type Job struct {
	AlbumID string
	Artist  string
	Album   string
	Year    int
}
