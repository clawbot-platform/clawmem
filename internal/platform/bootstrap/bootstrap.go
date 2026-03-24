package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"clawmem/internal/config"
	"clawmem/internal/domain/memory"
	"clawmem/internal/platform/store"
	memoryservice "clawmem/internal/services/memory"
	replayservice "clawmem/internal/services/replay"
	trustservice "clawmem/internal/services/trust"
)

type Dependencies struct {
	Store   *store.FileStore
	Memory  *memoryservice.Service
	Replay  *replayservice.Service
	Trust   *trustservice.Service
	ReadyFn func(context.Context) error
}

type seedFile struct {
	Memories []memory.MemoryRecord `json:"memories"`
}

func Build(ctx context.Context, cfg config.Config, logger *slog.Logger) (Dependencies, error) {
	fileStore, err := store.NewFileStore(cfg.StoragePath)
	if err != nil {
		return Dependencies{}, err
	}

	memorySvc := memoryservice.NewService(fileStore)
	replaySvc := replayservice.NewService(memorySvc)
	trustSvc := trustservice.NewService(memorySvc)

	if err := maybeSeed(ctx, cfg, memorySvc, logger); err != nil {
		return Dependencies{}, err
	}

	return Dependencies{
		Store:  fileStore,
		Memory: memorySvc,
		Replay: replaySvc,
		Trust:  trustSvc,
		ReadyFn: func(ctx context.Context) error {
			_, err := fileStore.Count(ctx)
			return err
		},
	}, nil
}

func maybeSeed(ctx context.Context, cfg config.Config, memorySvc *memoryservice.Service, logger *slog.Logger) error {
	if !cfg.SeedOnStartup {
		return nil
	}

	count, err := memorySvc.Count(ctx)
	if err != nil {
		return fmt.Errorf("count stored memories: %w", err)
	}
	if count > 0 {
		return nil
	}

	payload, err := os.ReadFile(cfg.SeedPath)
	if err != nil {
		return fmt.Errorf("read seed file: %w", err)
	}

	var seed seedFile
	if err := json.Unmarshal(payload, &seed); err != nil {
		return fmt.Errorf("decode seed file: %w", err)
	}

	for _, record := range seed.Memories {
		if _, err := memorySvc.CreateSeed(ctx, record); err != nil {
			return fmt.Errorf("seed memory record %s: %w", record.ID, err)
		}
	}

	logger.Info("seeded memory records", "count", len(seed.Memories), "seed_path", cfg.SeedPath)
	return nil
}
