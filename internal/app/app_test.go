package app

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/okarabudka/url-shortener/internal/config"
)

const (
	testUser     = "admin"
	testPassword = "secret"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()

	return &config.Config{
		Env:         config.EnvLocal,
		StoragePath: filepath.Join(t.TempDir(), "storage.db"),
		AliasLength: 6,
		HTTPServer: config.HTTPServer{
			Address:     "127.0.0.1:0",
			Timeout:     4 * time.Second,
			IdleTimeout: 60 * time.Second,
			User:        testUser,
			Password:    testPassword,
		},
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNew_UsesConfiguredTimeouts(t *testing.T) {
	cfg := testConfig(t)

	a, err := New(cfg, discardLogger())
	require.NoError(t, err)
	t.Cleanup(func() { _ = a.storage.Close() })

	assert.Equal(t, cfg.HTTPServer.Address, a.server.Addr)
	assert.Equal(t, cfg.HTTPServer.Timeout, a.server.ReadHeaderTimeout)
	assert.Equal(t, cfg.HTTPServer.Timeout, a.server.ReadTimeout)
	assert.Equal(t, cfg.HTTPServer.Timeout, a.server.WriteTimeout)
	assert.Equal(t, cfg.HTTPServer.IdleTimeout, a.server.IdleTimeout)
}

func TestRun_ReturnsNilAfterShutdown(t *testing.T) {
	a, err := New(testConfig(t), discardLogger())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	assert.NoError(t, a.Run(ctx))
}

func TestRun_FailsWhenAddressIsInUse(t *testing.T) {
	var lc net.ListenConfig
	busy, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = busy.Close() })

	cfg := testConfig(t)
	cfg.HTTPServer.Address = busy.Addr().String()

	a, err := New(cfg, discardLogger())
	require.NoError(t, err)

	err = a.Run(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listen on "+busy.Addr().String())
}

// TestEndToEnd walks a link through its whole life: create, follow, delete, follow again.
func TestEndToEnd(t *testing.T) {
	a, err := New(testConfig(t), discardLogger())
	require.NoError(t, err)
	t.Cleanup(func() { _ = a.storage.Close() })

	srv := httptest.NewServer(a.server.Handler)
	t.Cleanup(srv.Close)

	client := srv.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	type result struct {
		StatusCode int
		Location   string
	}

	do := func(method, path, body string, auth bool) result {
		t.Helper()

		req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, strings.NewReader(body))
		require.NoError(t, err)
		if auth {
			req.SetBasicAuth(testUser, testPassword)
		}

		res, err := client.Do(req)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())

		return result{StatusCode: res.StatusCode, Location: res.Header.Get("Location")}
	}

	link := `{"url":"https://example.com/some/long/path","alias":"example"}`

	res := do(http.MethodPost, "/url", link, false)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "create without auth")

	res = do(http.MethodPost, "/url", link, true)
	assert.Equal(t, http.StatusCreated, res.StatusCode, "create")

	res = do(http.MethodPost, "/url", link, true)
	assert.Equal(t, http.StatusConflict, res.StatusCode, "create with a taken alias")

	res = do(http.MethodGet, "/example", "", false)
	assert.Equal(t, http.StatusFound, res.StatusCode, "follow")
	assert.Equal(t, "https://example.com/some/long/path", res.Location)

	res = do(http.MethodDelete, "/url/example", "", false)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "delete without auth")

	res = do(http.MethodDelete, "/url/example", "", true)
	assert.Equal(t, http.StatusNoContent, res.StatusCode, "delete")

	res = do(http.MethodDelete, "/url/example", "", true)
	assert.Equal(t, http.StatusNotFound, res.StatusCode, "delete again")

	res = do(http.MethodGet, "/example", "", false)
	assert.Equal(t, http.StatusNotFound, res.StatusCode, "follow after delete")
}
