package cache

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type StateStore struct {
	DB *sql.DB
}

type ReviewState struct {
	ReviewID    string
	Repo        string
	BaseSHA     string
	HeadSHA     string
	Fingerprint string
	UpdatedAt   string
}

func Open(path string) (*StateStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(context.Background()); err != nil {
		return nil, err
	}
	store := &StateStore{DB: db}
	if err := store.ensureSchema(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *StateStore) ensureSchema() error {
	ddl := `
CREATE TABLE IF NOT EXISTS review_state (
  repo TEXT NOT NULL,
  review_id TEXT NOT NULL,
  base_sha TEXT NOT NULL,
  head_sha TEXT NOT NULL,
  fingerprint TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (repo, review_id, head_sha)
);
`
	_, err := s.DB.Exec(ddl)
	return err
}

func (s *StateStore) Upsert(ctx context.Context, state ReviewState) error {
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.ExecContext(ctx, `
	INSERT INTO review_state (repo, review_id, base_sha, head_sha, fingerprint, updated_at)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(repo, review_id, head_sha) DO UPDATE SET
	  base_sha=excluded.base_sha,
	  fingerprint=excluded.fingerprint,
	  updated_at=excluded.updated_at`,
		state.Repo, state.ReviewID, state.BaseSHA, state.HeadSHA, state.Fingerprint, state.UpdatedAt)
	return err
}

func (s *StateStore) Get(ctx context.Context, repo, reviewID, headSHA string) (ReviewState, error) {
	var row ReviewState
	err := s.DB.QueryRowContext(ctx, `
SELECT review_id, repo, base_sha, head_sha, fingerprint, updated_at
FROM review_state
WHERE repo = ? AND review_id = ? AND head_sha = ?`, repo, reviewID, headSHA).Scan(
		&row.ReviewID, &row.Repo, &row.BaseSHA, &row.HeadSHA, &row.Fingerprint, &row.UpdatedAt,
	)
	if err != nil {
		return row, fmt.Errorf("state miss: %w", err)
	}
	return row, nil
}
