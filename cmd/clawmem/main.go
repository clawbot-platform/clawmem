package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"clawmem/internal/app"
	"clawmem/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("build app: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
