package metadata

import (
	"context"

	"github.com/temic/go-music/internal/metadata/model"
)

// Provider fetches album metadata from an external or local source.
type Provider interface {
	Name() string
	SearchAlbum(ctx context.Context, query model.SearchQuery) (*model.Result, error)
}
