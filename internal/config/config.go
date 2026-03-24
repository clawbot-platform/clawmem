package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AppEnv        string
	LogLevel      string
	Addr          string
	StoragePath   string
	SeedPath      string
	SeedOnStartup bool
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:        getenv("CLAWMEM_ENV", "development"),
		LogLevel:      strings.ToLower(getenv("CLAWMEM_LOG_LEVEL", "info")),
		Addr:          getenv("CLAWMEM_ADDR", "127.0.0.1:8088"),
		StoragePath:   getenv("CLAWMEM_STORAGE_PATH", "./var/clawmem"),
		SeedPath:      getenv("CLAWMEM_SEED_PATH", "./configs/memory/seed-memory.json"),
		SeedOnStartup: parseBool(getenv("CLAWMEM_SEED_ON_STARTUP", "true")),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return errors.New("CLAWMEM_ADDR is required")
	}
	if strings.TrimSpace(c.StoragePath) == "" {
		return errors.New("CLAWMEM_STORAGE_PATH is required")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("CLAWMEM_LOG_LEVEL must be one of debug, info, warn, error")
	}
	return nil
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
