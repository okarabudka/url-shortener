package app

import (
	"context"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/okarabudka/url-shortener/internal/config"
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
