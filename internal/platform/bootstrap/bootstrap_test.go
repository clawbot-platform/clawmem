package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"clawmem/internal/config"
)

func TestBuildSeedsStoreWhenEnabledAndEmpty(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv:        "test",
		LogLevel:      "info",
		Addr:          "127.0.0.1:0",
		StoragePath:   filepath.Join(tempDir, "data"),
		SeedPath:      filepath.Join(tempDir, "seed.json"),
		SeedOnStartup: true,
	}
	seed := `{"memories":[{"id":"mem-seed-001","memory_type":"scenario_summary","scope":"platform","scenario_id":"sample-order-review","source_id":"sample-pack-001","summary":"Seed summary.","metadata":{"pack_id":"sample-pack"},"tags":["seed"],"created_at":"2026-03-24T00:00:00Z","updated_at":"2026-03-24T00:00:00Z"}]}`
	if err := os.WriteFile(cfg.SeedPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	deps, err := Build(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	count, err := deps.Memory.Count(context.Background())
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("expected seeded count 1, got %d", count)
	}
	if err := deps.ReadyFn(context.Background()); err != nil {
		t.Fatalf("ReadyFn() error = %v", err)
	}
	if deps.Ops == nil {
		t.Fatal("expected ops service dependency")
	}
}

func TestBuildDoesNotReseedExistingStore(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv:        "test",
		LogLevel:      "info",
		Addr:          "127.0.0.1:0",
		StoragePath:   filepath.Join(tempDir, "data"),
		SeedPath:      filepath.Join(tempDir, "seed.json"),
		SeedOnStartup: true,
	}
	seed := `{"memories":[{"id":"mem-seed-001","memory_type":"scenario_summary","scope":"platform","scenario_id":"sample-order-review","source_id":"sample-pack-001","summary":"Seed summary.","metadata":{"pack_id":"sample-pack"},"tags":["seed"],"created_at":"2026-03-24T00:00:00Z","updated_at":"2026-03-24T00:00:00Z"}]}`
	if err := os.WriteFile(cfg.SeedPath, []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	first, err := Build(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("Build(first) error = %v", err)
	}
	second, err := Build(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("Build(second) error = %v", err)
	}

	count, err := second.Memory.Count(context.Background())
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1 after second build, got %d", count)
	}
	_, _ = first, second
}
