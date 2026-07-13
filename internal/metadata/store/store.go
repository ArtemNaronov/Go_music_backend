package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/temic/go-music/internal/metadata/model"
)

// Store persists album enrichment metadata in SQLite.
type Store struct {
	db *sql.DB
}

// Open creates or opens the metadata database.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS album_metadata (
	album_id TEXT PRIMARY KEY,
	artist TEXT NOT NULL,
	title TEXT NOT NULL,
	year INTEGER NOT NULL DEFAULT 0,
	description TEXT NOT NULL DEFAULT '',
	genres TEXT NOT NULL DEFAULT '[]',
	musicbrainz_id TEXT NOT NULL DEFAULT '',
	cover_path TEXT NOT NULL DEFAULT '',
	fetch_status TEXT NOT NULL DEFAULT 'pending',
	fetched_at TEXT,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_album_metadata_status ON album_metadata(fetch_status);
`)
	return err
}

// Close closes the database.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Get returns stored metadata for an album.
func (s *Store) Get(ctx context.Context, albumID string) (model.AlbumRecord, bool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT album_id, artist, title, year, description, genres, musicbrainz_id, cover_path,
       fetch_status, fetched_at, updated_at
FROM album_metadata
WHERE album_id = ?
`, albumID)

	record, err := scanRecord(row)
	if err == sql.ErrNoRows {
		return model.AlbumRecord{}, false, nil
	}
	if err != nil {
		return model.AlbumRecord{}, false, err
	}

	return record, true, nil
}

// Save upserts album model.
func (s *Store) Save(ctx context.Context, record model.AlbumRecord) error {
	genresJSON, err := json.Marshal(record.Genres)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = now
	}

	var fetchedAt any
	if !record.FetchedAt.IsZero() {
		fetchedAt = record.FetchedAt.UTC().Format(time.RFC3339)
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO album_metadata (
	album_id, artist, title, year, description, genres, musicbrainz_id, cover_path,
	fetch_status, fetched_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(album_id) DO UPDATE SET
	artist = excluded.artist,
	title = excluded.title,
	year = excluded.year,
	description = excluded.description,
	genres = excluded.genres,
	musicbrainz_id = excluded.musicbrainz_id,
	cover_path = excluded.cover_path,
	fetch_status = excluded.fetch_status,
	fetched_at = excluded.fetched_at,
	updated_at = excluded.updated_at
`,
		record.AlbumID,
		record.Artist,
		record.Title,
		record.Year,
		record.Description,
		string(genresJSON),
		record.MusicBrainzID,
		record.CoverPath,
		string(record.FetchStatus),
		fetchedAt,
		record.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// ShouldFetch reports whether a background fetch should run for the album.
func (s *Store) ShouldFetch(ctx context.Context, albumID string) (bool, error) {
	record, ok, err := s.Get(ctx, albumID)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}

	switch record.FetchStatus {
	case model.StatusSuccess, model.StatusSkipped:
		return false, nil
	case model.StatusFailed:
		return time.Since(record.UpdatedAt) > 24*time.Hour, nil
	default:
		return true, nil
	}
}

// MarkPending creates or resets a pending record for an album.
func (s *Store) MarkPending(ctx context.Context, albumID, artist, title string, year int) error {
	now := time.Now().UTC()
	return s.Save(ctx, model.AlbumRecord{
		AlbumID:     albumID,
		Artist:      artist,
		Title:       title,
		Year:        year,
		Genres:      []string{},
		FetchStatus: model.StatusPending,
		UpdatedAt:   now,
	})
}

func scanRecord(row *sql.Row) (model.AlbumRecord, error) {
	var record model.AlbumRecord
	var genresJSON string
	var fetchStatus string
	var fetchedAt sql.NullString
	var updatedAt string

	err := row.Scan(
		&record.AlbumID,
		&record.Artist,
		&record.Title,
		&record.Year,
		&record.Description,
		&genresJSON,
		&record.MusicBrainzID,
		&record.CoverPath,
		&fetchStatus,
		&fetchedAt,
		&updatedAt,
	)
	if err != nil {
		return model.AlbumRecord{}, err
	}

	record.FetchStatus = model.FetchStatus(fetchStatus)
	if fetchedAt.Valid && fetchedAt.String != "" {
		if t, err := time.Parse(time.RFC3339, fetchedAt.String); err == nil {
			record.FetchedAt = t
		}
	}
	if updatedAt != "" {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			record.UpdatedAt = t
		}
	}

	genresJSON = strings.TrimSpace(genresJSON)
	if genresJSON == "" {
		record.Genres = []string{}
	} else if err := json.Unmarshal([]byte(genresJSON), &record.Genres); err != nil {
		record.Genres = []string{}
	}

	return record, nil
}
