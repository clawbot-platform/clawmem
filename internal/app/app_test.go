package app

import (
	"context"
	"path/filepath"
	"testing"

	"clawmem/internal/config"
)

func TestNewBuildsHTTPServer(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		AppEnv:        "test",
		LogLevel:      "debug",
		Addr:          "127.0.0.1:0",
		StoragePath:   filepath.Join(t.TempDir(), "clawmem"),
		SeedPath:      filepath.Join(t.TempDir(), "seed.json"),
		SeedOnStartup: false,
	}

	app, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if app.server == nil || app.server.Addr != cfg.Addr {
		t.Fatalf("unexpected app server %#v", app.server)
	}
}

func TestNewLoggerHonorsDebugLevel(t *testing.T) {
	t.Parallel()

	logger := newLogger("debug")
	handler := logger.Handler()
	if !handler.Enabled(context.Background(), -4) {
		t.Fatal("expected debug level to be enabled")
	}
}
