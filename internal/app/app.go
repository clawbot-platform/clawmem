package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"clawmem/internal/config"
	"clawmem/internal/http/handlers"
	"clawmem/internal/http/routes"
	"clawmem/internal/platform/bootstrap"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
	server *http.Server
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger := newLogger(cfg.LogLevel)
	deps, err := bootstrap.Build(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	systemHandler := handlers.NewSystemHandler(deps.ReadyFn)
	memoryHandler := handlers.NewMemoryHandler(deps.Memory, deps.Replay, deps.Trust)

	return &App{
		cfg:    cfg,
		logger: logger,
		server: &http.Server{
			Addr:              cfg.Addr,
			Handler:           routes.New(systemHandler, memoryHandler, logger),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("starting clawmem", "addr", a.cfg.Addr, "env", a.cfg.AppEnv, "storage_path", a.cfg.StoragePath)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		a.logger.Info("shutting down clawmem")
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil
	}
}

func newLogger(level string) *slog.Logger {
	var handlerLevel slog.Level
	switch level {
	case "debug":
		handlerLevel = slog.LevelDebug
	case "warn":
		handlerLevel = slog.LevelWarn
	case "error":
		handlerLevel = slog.LevelError
	default:
		handlerLevel = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: handlerLevel}))
}
