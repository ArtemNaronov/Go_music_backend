package library

import (
	"context"
)

type coverData struct {
	data     []byte
	mimeType string
}

func (s *Store) SetCover(_ context.Context, trackID string, data []byte, mimeType string) {
	if len(data) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.covers == nil {
		s.covers = make(map[string]coverData)
	}

	s.covers[trackID] = coverData{
		data:     append([]byte(nil), data...),
		mimeType: mimeType,
	}
}

func (s *Store) GetCover(_ context.Context, trackID string) ([]byte, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cover, ok := s.covers[trackID]
	if !ok || len(cover.data) == 0 {
		return nil, "", false
	}

	return append([]byte(nil), cover.data...), cover.mimeType, true
}

func (s *Store) RemoveCover(_ context.Context, trackID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.covers, trackID)
}

func (s *Store) ClearCovers(_ context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.covers = make(map[string]coverData)
}

func (s *Store) clearCoversLocked() {
	s.covers = make(map[string]coverData)
}

var _ interface {
	SetCover(context.Context, string, []byte, string)
	GetCover(context.Context, string) ([]byte, string, bool)
	RemoveCover(context.Context, string)
	ClearCovers(context.Context)
} = (*Store)(nil)
