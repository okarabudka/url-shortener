// Package app wires storage, handlers and the HTTP server together and runs them.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/okarabudka/url-shortener/internal/config"
	"github.com/okarabudka/url-shortener/internal/http/handlers/redirect"
	"github.com/okarabudka/url-shortener/internal/http/handlers/save"
	mwLogger "github.com/okarabudka/url-shortener/internal/http/middleware/logger"
	"github.com/okarabudka/url-shortener/internal/lib/logger/sl"
	"github.com/okarabudka/url-shortener/internal/storage/sqlite"
)

// shutdownTimeout bounds how long in-flight requests may take after a stop signal.
const shutdownTimeout = 10 * time.Second

// URLStorage is everything the HTTP handlers need from the storage.
type URLStorage interface {
	save.URLSaver
	redirect.URLGetter
}

// App is the assembled service.
type App struct {
	log     *slog.Logger
	storage *sqlite.Storage
	server  *http.Server
}

// New opens the storage and builds the HTTP server from the configuration.
func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	st, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	return &App{
		log:     log,
		storage: st,
		server: &http.Server{
			Addr:              cfg.HTTPServer.Address,
			Handler:           NewRouter(log, st, cfg),
			ReadHeaderTimeout: cfg.HTTPServer.Timeout,
			ReadTimeout:       cfg.HTTPServer.Timeout,
			WriteTimeout:      cfg.HTTPServer.Timeout,
			IdleTimeout:       cfg.HTTPServer.IdleTimeout,
		},
	}, nil
}

// NewRouter registers the routes.
func NewRouter(log *slog.Logger, st URLStorage, cfg *config.Config) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)

	router.Post("/url", save.New(log, st, cfg.AliasLength))
	router.Get("/{alias}", redirect.New(log, st))

	return router
}

// Run serves HTTP until ctx is cancelled, then shuts the server down gracefully and closes the storage.
// It returns an error if the server cannot listen or stops for any other reason than the shutdown.
func (a *App) Run(ctx context.Context) error {
	defer func() {
		if err := a.storage.Close(); err != nil {
			a.log.Error("failed to close storage", sl.Err(err))
		}
	}()

	ln, err := net.Listen("tcp", a.server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", a.server.Addr, err)
	}

	a.log.Info("server started", slog.String("address", ln.Addr().String()))

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- a.server.Serve(ln)
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	a.log.Info("stopping server")

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	a.log.Info("server stopped")

	return nil
}
