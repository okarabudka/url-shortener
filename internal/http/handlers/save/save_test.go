package save_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/okarabudka/url-shortener/internal/http/handlers/save"
	"github.com/okarabudka/url-shortener/internal/storage"
)

type fakeSaver struct {
	err   error
	url   string
	alias string
	calls int
}

func (f *fakeSaver) SaveURL(urlToSave, alias string) (int64, error) {
	f.calls++
	f.url, f.alias = urlToSave, alias
	if f.err != nil {
		return 0, f.err
	}

	return 1, nil
}

const aliasLength = 7

func TestSave(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		saveErr    error
		wantStatus int
		wantError  string
		wantSaved  bool
	}{
		{name: "empty body", body: "", wantStatus: http.StatusBadRequest, wantError: "empty request"},
		{name: "broken json", body: `{"url":`, wantStatus: http.StatusBadRequest, wantError: "failed to decode request"},
		{name: "missing url", body: `{"alias":"example"}`, wantStatus: http.StatusBadRequest, wantError: "field URL is a required field"},
		{name: "javascript url", body: `{"url":"javascript:alert(1)"}`, wantStatus: http.StatusBadRequest, wantError: "field URL is not a valid http(s) URL"},
		{name: "not a url", body: `{"url":"example"}`, wantStatus: http.StatusBadRequest, wantError: "field URL is not a valid http(s) URL"},
		{name: "alias with slash", body: `{"url":"https://example.com","alias":"a/b"}`, wantStatus: http.StatusBadRequest, wantError: "field Alias may only contain letters and digits"},
		{name: "alias too long", body: `{"url":"https://example.com","alias":"` + strings.Repeat("a", 33) + `"}`, wantStatus: http.StatusBadRequest, wantError: "field Alias must be at most 32 characters long"},
		{name: "alias taken", body: `{"url":"https://example.com","alias":"example"}`, saveErr: storage.ErrURLExists, wantStatus: http.StatusConflict, wantError: "alias already exists", wantSaved: true},
		{name: "storage error", body: `{"url":"https://example.com","alias":"example"}`, saveErr: errors.New("disk on fire"), wantStatus: http.StatusInternalServerError, wantError: "failed to add url", wantSaved: true},
		{name: "saved", body: `{"url":"https://example.com","alias":"example"}`, wantStatus: http.StatusCreated, wantSaved: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saver := &fakeSaver{err: tc.saveErr}
			rec := serve(t, saver, tc.body)

			require.Equal(t, tc.wantStatus, rec.Code)

			var body save.Response
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tc.wantError, body.Error)
			assert.Equal(t, tc.wantSaved, saver.calls == 1)

			if tc.wantStatus == http.StatusCreated {
				assert.Equal(t, "OK", body.Status)
				assert.Equal(t, "example", body.Alias)
				assert.Equal(t, "https://example.com", saver.url)
			}
		})
	}
}

func TestSave_GeneratesAliasWhenMissing(t *testing.T) {
	saver := &fakeSaver{}
	rec := serve(t, saver, `{"url":"https://example.com"}`)

	require.Equal(t, http.StatusCreated, rec.Code)

	var body save.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body.Alias, aliasLength)
	assert.Equal(t, body.Alias, saver.alias)
}

func serve(t *testing.T, saver save.URLSaver, body string) *httptest.ResponseRecorder {
	t.Helper()

	handler := save.New(slog.New(slog.NewTextHandler(io.Discard, nil)), saver, aliasLength)

	req := httptest.NewRequest(http.MethodPost, "/url", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}
