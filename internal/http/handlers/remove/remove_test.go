package remove_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/okarabudka/url-shortener/internal/http/handlers/remove"
	"github.com/okarabudka/url-shortener/internal/storage"
)

type fakeDeleter struct {
	err     error
	deleted string
}

func (f *fakeDeleter) DeleteURL(alias string) error {
	f.deleted = alias
	return f.err
}

func TestRemove(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "deleted", wantStatus: http.StatusNoContent},
		{name: "not found", err: storage.ErrURLNotFound, wantStatus: http.StatusNotFound},
		{name: "storage error", err: errors.New("disk on fire"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deleter := &fakeDeleter{err: tc.err}
			router := chi.NewRouter()
			router.Delete("/url/{alias}", remove.New(slog.New(slog.NewTextHandler(io.Discard, nil)), deleter))

			req := httptest.NewRequest(http.MethodDelete, "/url/example", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			assert.Equal(t, "example", deleter.deleted)
		})
	}
}
