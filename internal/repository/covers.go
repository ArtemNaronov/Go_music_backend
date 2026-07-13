package repository

import "context"

// CoverStore stores embedded album artwork in memory.
type CoverStore interface {
	SetCover(ctx context.Context, trackID string, data []byte, mimeType string)
	GetCover(ctx context.Context, trackID string) ([]byte, string, bool)
	RemoveCover(ctx context.Context, trackID string)
	ClearCovers(ctx context.Context)
}
