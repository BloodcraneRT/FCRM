// Package watchlists stores local app metadata — watchlist membership,
// thesis note index — in SQLite. This is deliberately NOT time-series data:
// it's "what am I tracking and why," not "what happened when." Time-series
// signal/price/event history all lives in QuestDB.
package watchlists

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("watchlists: create data dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("watchlists: open %s: %w", path, err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS watchlist_items (
		symbol TEXT NOT NULL,
		list_name TEXT NOT NULL DEFAULT 'default',
		vertical TEXT,
		note TEXT,
		added_at TEXT NOT NULL,
		PRIMARY KEY (symbol, list_name)
	);
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS thesis_index (
		thesis_id TEXT PRIMARY KEY,
		symbol TEXT NOT NULL,
		title TEXT NOT NULL,
		file_path TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("watchlists: migrate: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

type Item struct {
	Symbol   string
	ListName string
	Vertical string
	Note     string
	AddedAt  time.Time
}

func (s *Store) Add(ctx context.Context, item Item) error {
	if item.ListName == "" {
		item.ListName = "default"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO watchlist_items (symbol, list_name, vertical, note, added_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(symbol, list_name) DO UPDATE SET vertical=excluded.vertical, note=excluded.note`,
		item.Symbol, item.ListName, item.Vertical, item.Note, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) Remove(ctx context.Context, symbol, listName string) error {
	if listName == "" {
		listName = "default"
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM watchlist_items WHERE symbol = ? AND list_name = ?`, symbol, listName)
	return err
}

// IndexThesis records a thesis markdown file in the local index so the CLI
// and dashboard can list thesis notes without scanning the filesystem.
func (s *Store) IndexThesis(thesisID, symbol, title, filePath string) error {
	_, err := s.db.Exec(
		`INSERT INTO thesis_index (thesis_id, symbol, title, file_path, created_at) VALUES (?, ?, ?, ?, ?)`,
		thesisID, symbol, title, filePath, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) List(ctx context.Context, listName string) ([]Item, error) {
	if listName == "" {
		listName = "default"
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT symbol, list_name, vertical, note, added_at FROM watchlist_items WHERE list_name = ? ORDER BY symbol`,
		listName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var it Item
		var addedAt string
		if err := rows.Scan(&it.Symbol, &it.ListName, &it.Vertical, &it.Note, &addedAt); err != nil {
			return nil, err
		}
		it.AddedAt, _ = time.Parse(time.RFC3339, addedAt)
		items = append(items, it)
	}
	return items, rows.Err()
}
