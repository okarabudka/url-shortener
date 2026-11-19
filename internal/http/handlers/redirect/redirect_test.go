package redirect_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/okarabudka/url-shortener/internal/http/handlers/redirect"
	"github.com/okarabudka/url-shortener/internal/storage"
)

type fakeGetter struct {
	url string
	err error
}

func (f fakeGetter) GetURL(string) (string, error) { return f.url, f.err }

func TestRedirect(t *testing.T) {
	cases := []struct {
		name         string
		getter       fakeGetter
		wantStatus   int
		wantLocation string
	}{
		{name: "found", getter: fakeGetter{url: "https://example.com"}, wantStatus: http.StatusFound, wantLocation: "https://example.com"},
		{name: "not found", getter: fakeGetter{err: storage.ErrURLNotFound}, wantStatus: http.StatusNotFound},
		{name: "storage error", getter: fakeGetter{err: errors.New("disk on fire")}, wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := chi.NewRouter()
			router.Get("/{alias}", redirect.New(slog.New(slog.NewTextHandler(io.Discard, nil)), tc.getter))

			req := httptest.NewRequest(http.MethodGet, "/example", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)
			assert.Equal(t, tc.wantLocation, rec.Header().Get("Location"))
		})
	}
}
