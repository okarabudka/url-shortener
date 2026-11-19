// Package sqlite stores links in a SQLite database.
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"

	"github.com/okarabudka/url-shortener/internal/storage"
)

// Storage is a SQLite-backed link store.
type Storage struct {
	db *sql.DB
}

// New opens the database at storagePath and creates the table if needed.
func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// UNIQUE already indexes alias, so no separate index is needed.
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS url(
		id INTEGER PRIMARY KEY,
		alias TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL)`)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("%s: %w", op, err), db.Close())
	}

	return &Storage{db: db}, nil
}

// Close closes the database.
func (s *Storage) Close() error {
	return s.db.Close()
}

// SaveURL stores urlToSave under alias and returns its id, or storage.ErrURLExists if the alias is taken.
func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	res, err := s.db.Exec("INSERT INTO url(url, alias) VALUES(?, ?)", urlToSave, alias)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

// GetURL returns the URL stored under alias, or storage.ErrURLNotFound.
func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	var resURL string

	err := s.db.QueryRow("SELECT url FROM url WHERE alias = ?", alias).Scan(&resURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return resURL, nil
}

// DeleteURL deletes the link stored under alias, or returns storage.ErrURLNotFound.
func (s *Storage) DeleteURL(alias string) error {
	const op = "storage.sqlite.DeleteURL"

	res, err := s.db.Exec("DELETE FROM url WHERE alias = ?", alias)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: failed to get affected rows: %w", op, err)
	}
	if affected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
	}

	return nil
}
