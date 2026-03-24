package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("CLAWMEM_ENV", "")
	t.Setenv("CLAWMEM_LOG_LEVEL", "")
	t.Setenv("CLAWMEM_ADDR", "")
	t.Setenv("CLAWMEM_STORAGE_PATH", "")
	t.Setenv("CLAWMEM_SEED_PATH", "")
	t.Setenv("CLAWMEM_SEED_ON_STARTUP", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Addr != "127.0.0.1:8088" {
		t.Fatalf("expected default addr, got %q", cfg.Addr)
	}
	if !cfg.SeedOnStartup {
		t.Fatalf("expected seeding enabled by default")
	}
}

func TestValidateRejectsBadLogLevel(t *testing.T) {
	cfg := Config{
		AppEnv:      "development",
		LogLevel:    "verbose",
		Addr:        "127.0.0.1:8088",
		StoragePath: "./var/clawmem",
		SeedPath:    "./configs/memory/seed-memory.json",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
