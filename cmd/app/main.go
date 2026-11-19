// Command app runs the url-shortener HTTP service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/okarabudka/url-shortener/internal/app"
	"github.com/okarabudka/url-shortener/internal/config"
	"github.com/okarabudka/url-shortener/internal/lib/logger/sl"
)

func main() {
	cfg, err := config.Load(config.Path())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	log := newLogger(cfg.Env)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", sl.Err(err))
		os.Exit(1)
	}
}

func run(cfg *config.Config, log *slog.Logger) error {
	log.Info("starting url-shortener", slog.String("env", cfg.Env))

	application, err := app.New(cfg, log)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return application.Run(ctx)
}

func newLogger(env string) *slog.Logger {
	level := slog.LevelDebug
	if env == config.EnvProd {
		level = slog.LevelInfo
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
