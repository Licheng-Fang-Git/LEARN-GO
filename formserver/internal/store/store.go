// Package store persists form submissions in SQLite using the standard
// database/sql interface. Each submission's field values are stored as a JSON
// blob in a single column, which lets one table hold responses for ANY form
// definition without a schema migration per form.
//
// The only non-stdlib dependency in the whole project lives here: a SQL driver.
// Go's database/sql is an interface that needs a concrete driver registered
// under the hood. We use modernc.org/sqlite, a pure-Go SQLite (no CGO, no C
// compiler required), so `go build` stays trivially portable.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// driverName is set by whichever driver file is compiled in (see
// driver_modernc.go / driver_mattn.go). The default build uses the pure-Go
// modernc.org/sqlite driver; build with -tags cgosqlite to use mattn/go-sqlite3.

// ErrNotFound is returned when a submission id doesn't exist.
var ErrNotFound = errors.New("submission not found")

// Submission is one stored form response.
type Submission struct {
	ID        int64          `json:"id"`
	FormID    string         `json:"form_id"`
	Data      map[string]any `json:"data"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// Store wraps the database handle.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database file at path.
func Open(path string) (*Store, error) {
	db, err := sql.Open(driverName, path)
	if err != nil {
		return nil, err
	}
	// SQLite handles one writer at a time; capping the pool at a single
	// connection keeps this demo free of "database is locked" errors.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// Migrate creates the submissions table if it doesn't exist. Timestamps are
// stored as RFC3339 text for portable, driver-independent round-tripping.
func (s *Store) Migrate(ctx context.Context) error {
	const q = `
CREATE TABLE IF NOT EXISTS submissions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    form_id    TEXT NOT NULL,
    data       TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_submissions_form ON submissions(form_id);`
	_, err := s.db.ExecContext(ctx, q)
	return err
}

const tsLayout = time.RFC3339Nano

// Create inserts a new submission and returns it with its assigned id. (CRUD: C)
func (s *Store) Create(ctx context.Context, formID string, data map[string]any) (*Submission, error) {
	blob, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO submissions(form_id, data, created_at, updated_at) VALUES(?, ?, ?, ?)`,
		formID, string(blob), now.Format(tsLayout), now.Format(tsLayout))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Submission{ID: id, FormID: formID, Data: data, CreatedAt: now, UpdatedAt: now}, nil
}

// Get fetches a single submission by id. (CRUD: R)
func (s *Store) Get(ctx context.Context, id int64) (*Submission, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, form_id, data, created_at, updated_at FROM submissions WHERE id = ?`, id)
	sub, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return sub, err
}

// ListByForm returns every submission for a form, newest first. (CRUD: R)
func (s *Store) ListByForm(ctx context.Context, formID string) ([]*Submission, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, form_id, data, created_at, updated_at FROM submissions WHERE form_id = ? ORDER BY id DESC`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Submission
	for rows.Next() {
		sub, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// Update replaces the data of an existing submission. (CRUD: U)
func (s *Store) Update(ctx context.Context, id int64, data map[string]any) (*Submission, error) {
	blob, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`UPDATE submissions SET data = ?, updated_at = ? WHERE id = ?`,
		string(blob), now.Format(tsLayout), id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

// Delete removes a submission by id. (CRUD: D)
func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM submissions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scan(sc scanner) (*Submission, error) {
	var (
		sub              Submission
		blob             string
		created, updated string
	)
	if err := sc.Scan(&sub.ID, &sub.FormID, &blob, &created, &updated); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(blob), &sub.Data); err != nil {
		return nil, err
	}
	sub.CreatedAt, _ = time.Parse(tsLayout, created)
	sub.UpdatedAt, _ = time.Parse(tsLayout, updated)
	return &sub, nil
}
