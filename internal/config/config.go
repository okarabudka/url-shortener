// Package config loads the service configuration from a YAML file with environment overrides.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Supported values of Config.Env.
const (
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"
)

// DefaultPath is used when CONFIG_PATH is not set. It is relative to the working directory.
const DefaultPath = "configs/config.yaml"

// Config is the service configuration.
type Config struct {
	Env         string     `yaml:"env" env-default:"local"`
	StoragePath string     `yaml:"db_storage_path" env:"STORAGE_PATH"`
	AliasLength int        `yaml:"alias_length" env-default:"6"`
	HTTPServer  HTTPServer `yaml:"http"`
}

// HTTPServer configures the HTTP listener and the Basic Auth credentials.
type HTTPServer struct {
	Address     string        `yaml:"address" env:"HTTP_ADDRESS" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
	// Credentials are read from the environment only, so they never end up in a committed file.
	User     string `yaml:"-" env:"HTTP_USER"`
	Password string `yaml:"-" env:"HTTP_PASSWORD"`
}

// Path returns the config file path from CONFIG_PATH or DefaultPath.
func Path() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	return DefaultPath
}

// Load reads the YAML file at path, applies environment overrides and validates the result.
func Load(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	var errs []error

	switch c.Env {
	case EnvLocal, EnvDev, EnvProd:
	default:
		errs = append(errs, fmt.Errorf("env %q is unknown, use %s, %s or %s", c.Env, EnvLocal, EnvDev, EnvProd))
	}
	if c.StoragePath == "" {
		errs = append(errs, errors.New("db_storage_path (or STORAGE_PATH) is empty"))
	}
	if c.AliasLength < 1 {
		errs = append(errs, fmt.Errorf("alias_length must be positive, got %d", c.AliasLength))
	}
	if c.HTTPServer.Timeout <= 0 || c.HTTPServer.IdleTimeout <= 0 {
		errs = append(errs, errors.New("http.timeout and http.idle_timeout must be positive durations such as 4s"))
	}
	// Closed by default: without credentials anyone could create and delete links.
	if c.HTTPServer.User == "" || c.HTTPServer.Password == "" {
		errs = append(errs, errors.New("HTTP_USER and HTTP_PASSWORD must be set, they protect creating and deleting links"))
	}

	return errors.Join(errs...)
}
