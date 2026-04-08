package app

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

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

func TestNewLoggerHonorsWarnAndDefaultLevels(t *testing.T) {
	t.Parallel()

	if !newLogger("warn").Handler().Enabled(context.Background(), 4) {
		t.Fatal("expected warn level to allow warn logs")
	}
	if !newLogger("info").Handler().Enabled(context.Background(), 0) {
		t.Fatal("expected default info level to allow info logs")
	}
}

func TestRunShutsDownOnContextCancel(t *testing.T) {
	t.Parallel()

	// Some restricted sandboxes disallow binding sockets entirely.
	preflightListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if isSocketBindPermissionError(err) {
			t.Skipf("skipping bind-dependent test in restricted environment: %v", err)
		}
		t.Fatalf("preflight listen error = %v", err)
	}
	_ = preflightListener.Close()

	cfg := config.Config{
		AppEnv:        "test",
		LogLevel:      "info",
		Addr:          "127.0.0.1:0",
		StoragePath:   filepath.Join(t.TempDir(), "clawmem"),
		SeedPath:      filepath.Join(t.TempDir(), "seed.json"),
		SeedOnStartup: false,
	}

	app, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			if isSocketBindPermissionError(err) {
				t.Skipf("skipping bind-dependent test in restricted environment: %v", err)
			}
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run() to return")
	}
}

func isSocketBindPermissionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "operation not permitted") ||
		strings.Contains(message, "permission denied")
}
