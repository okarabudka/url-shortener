package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/okarabudka/url-shortener/internal/storage"
	"github.com/okarabudka/url-shortener/internal/storage/sqlite"
)

// newStorage opens a fresh database file per test. A file in t.TempDir behaves like production
// and avoids the pitfalls of :memory: with the database/sql connection pool.
func newStorage(t *testing.T) *sqlite.Storage {
	t.Helper()

	st, err := sqlite.New(filepath.Join(t.TempDir(), "storage.db"))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, st.Close()) })

	return st
}

func TestStorage(t *testing.T) {
	const (
		alias = "example"
		url   = "https://example.com"
	)

	cases := []struct {
		name string
		run  func(t *testing.T, st *sqlite.Storage)
	}{
		{"save and get", func(t *testing.T, st *sqlite.Storage) {
			id, err := st.SaveURL(url, alias)
			require.NoError(t, err)
			assert.Positive(t, id)

			got, err := st.GetURL(alias)
			require.NoError(t, err)
			assert.Equal(t, url, got)
		}},
		{"save with a taken alias", func(t *testing.T, st *sqlite.Storage) {
			_, err := st.SaveURL(url, alias)
			require.NoError(t, err)

			_, err = st.SaveURL("https://other.example.com", alias)
			assert.ErrorIs(t, err, storage.ErrURLExists)
		}},
		{"get a missing alias", func(t *testing.T, st *sqlite.Storage) {
			_, err := st.GetURL("missing")
			assert.ErrorIs(t, err, storage.ErrURLNotFound)
		}},
		{"delete", func(t *testing.T, st *sqlite.Storage) {
			_, err := st.SaveURL(url, alias)
			require.NoError(t, err)

			require.NoError(t, st.DeleteURL(alias))

			_, err = st.GetURL(alias)
			assert.ErrorIs(t, err, storage.ErrURLNotFound)
		}},
		{"delete a missing alias", func(t *testing.T, st *sqlite.Storage) {
			assert.ErrorIs(t, st.DeleteURL("missing"), storage.ErrURLNotFound)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t, newStorage(t))
		})
	}
}

func TestNew_KeepsDataAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "storage.db")

	st, err := sqlite.New(path)
	require.NoError(t, err)
	_, err = st.SaveURL("https://example.com", "example")
	require.NoError(t, err)
	require.NoError(t, st.Close())

	st, err = sqlite.New(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })

	got, err := st.GetURL("example")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", got)
}
