package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/okarabudka/url-shortener/internal/config"
)

// setEnv isolates a test from the caller's environment: every variable the config reads is
// unset first, then the given ones are set. The originals come back after the test.
func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()

	for _, name := range []string{"CONFIG_PATH", "HTTP_ADDRESS", "STORAGE_PATH", "HTTP_USER", "HTTP_PASSWORD"} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	for name, value := range vars {
		t.Setenv(name, value)
	}
}

func credentials() map[string]string {
	return map[string]string{"HTTP_USER": "admin", "HTTP_PASSWORD": "secret"}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestLoad_ReadsHTTPSectionFromFile(t *testing.T) {
	setEnv(t, credentials())

	cfg, err := config.Load("testdata/config.yaml")
	require.NoError(t, err)

	assert.Equal(t, config.EnvDev, cfg.Env)
	assert.Equal(t, "/tmp/test.db", cfg.StoragePath)
	assert.Equal(t, 8, cfg.AliasLength)
	assert.Equal(t, "127.0.0.1:9123", cfg.HTTPServer.Address)
	assert.Equal(t, 4*time.Second, cfg.HTTPServer.Timeout)
	assert.Equal(t, 90*time.Second, cfg.HTTPServer.IdleTimeout)
	assert.Equal(t, "admin", cfg.HTTPServer.User)
	assert.Equal(t, "secret", cfg.HTTPServer.Password)
}

func TestLoad_AppliesDefaults(t *testing.T) {
	setEnv(t, credentials())

	cfg, err := config.Load(writeConfig(t, "db_storage_path: ./storage.db\n"))
	require.NoError(t, err)

	assert.Equal(t, config.EnvLocal, cfg.Env)
	assert.Equal(t, 6, cfg.AliasLength)
	assert.Equal(t, "localhost:8080", cfg.HTTPServer.Address)
	assert.Equal(t, 4*time.Second, cfg.HTTPServer.Timeout)
	assert.Equal(t, 60*time.Second, cfg.HTTPServer.IdleTimeout)
}

func TestLoad_EnvironmentOverridesFile(t *testing.T) {
	vars := credentials()
	vars["HTTP_ADDRESS"] = "0.0.0.0:9999"
	vars["STORAGE_PATH"] = "/data/other.db"
	setEnv(t, vars)

	cfg, err := config.Load("testdata/config.yaml")
	require.NoError(t, err)

	assert.Equal(t, "0.0.0.0:9999", cfg.HTTPServer.Address)
	assert.Equal(t, "/data/other.db", cfg.StoragePath)
}

func TestLoad_Errors(t *testing.T) {
	cases := []struct {
		name    string
		config  string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "unknown env",
			config:  "env: foo\ndb_storage_path: ./storage.db\n",
			env:     credentials(),
			wantErr: `env "foo" is unknown`,
		},
		{
			name:    "missing password",
			config:  "db_storage_path: ./storage.db\n",
			env:     map[string]string{"HTTP_USER": "admin"},
			wantErr: "HTTP_USER and HTTP_PASSWORD must be set",
		},
		{
			name:    "missing storage path",
			config:  "env: local\n",
			env:     credentials(),
			wantErr: "db_storage_path (or STORAGE_PATH) is empty",
		},
		{
			name:    "timeout without unit",
			config:  "db_storage_path: ./storage.db\nhttp:\n  timeout: 4\n",
			env:     credentials(),
			wantErr: "into time.Duration",
		},
		{
			name:    "negative timeout",
			config:  "db_storage_path: ./storage.db\nhttp:\n  idle_timeout: -1s\n",
			env:     credentials(),
			wantErr: "must be positive durations",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.env)

			_, err := config.Load(writeConfig(t, tc.config))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestLoad_MissingFile(t *testing.T) {
	setEnv(t, credentials())

	_, err := config.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	assert.Error(t, err)
}

func TestPath(t *testing.T) {
	setEnv(t, nil)
	assert.Equal(t, config.DefaultPath, config.Path())

	t.Setenv("CONFIG_PATH", "/etc/url-shortener/config.yaml")
	assert.Equal(t, "/etc/url-shortener/config.yaml", config.Path())
}
