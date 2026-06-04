package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Tarasa24/psp-integration-demo/internal/processor"
	"github.com/Tarasa24/psp-integration-demo/internal/provider"
	"github.com/Tarasa24/psp-integration-demo/internal/providerfactory"
	"github.com/Tarasa24/psp-integration-demo/internal/server"
	"github.com/Tarasa24/psp-integration-demo/internal/storage"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx := context.Background()

	dsn := mustEnv("DATABASE_URL")
	migrationsDir := envOr("MIGRATIONS_DIR", "./migrations")

	pool, err := storage.NewPool(ctx, dsn, migrationsDir)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	chargeRepo := storage.NewChargeRepository(pool)
	webhookRepo := storage.NewWebhookRepository(pool)
	idempotencyRepo := storage.NewIdempotencyRepository(pool)

	activeProvider := envOr("ACTIVE_PROVIDER", "mock")

	cfg := providerfactory.Config{
		StripeAPIKey:        os.Getenv("STRIPE_API_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		MockWebhookSecret:   envOr("MOCK_WEBHOOK_SECRET", "mock-secret"),
	}

	prov, err := providerfactory.NewProvider(activeProvider, cfg)
	if err != nil {
		slog.Error("failed to initialise provider", "error", err)
		os.Exit(1)
	}

	providers := map[string]provider.Provider{
		activeProvider: prov,
	}

	// Always register mock for webhook testing convenience.
	if activeProvider != "mock" {
		mockProv, _ := providerfactory.NewProvider("mock", cfg)
		providers["mock"] = mockProv
	}

	proc := processor.New(chargeRepo, webhookRepo, 5*time.Second)
	procCtx, procCancel := context.WithCancel(ctx)
	defer procCancel()
	go proc.Run(procCtx)

	addr := envOr("SERVER_ADDR", ":8080")
	srv := server.NewServer(server.Deps{
		Providers:       providers,
		ChargeRepo:      chargeRepo,
		WebhookRepo:     webhookRepo,
		IdempotencyRepo: idempotencyRepo,
		ActiveProvider:  activeProvider,
		Addr:            addr,
	})

	if err := server.Run(srv); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
