package server

import (
	"context"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tarasa24/psp-integration-demo/internal/apidocs"
	"github.com/Tarasa24/psp-integration-demo/internal/provider"
	"github.com/Tarasa24/psp-integration-demo/internal/repository"
	"github.com/Tarasa24/psp-integration-demo/internal/server/handlers"
	"github.com/Tarasa24/psp-integration-demo/internal/server/middleware"
)

// Deps holds all dependencies injected into the HTTP server.
type Deps struct {
	Providers       map[string]provider.Provider
	ChargeRepo      repository.ChargeRepository
	WebhookRepo     repository.WebhookRepository
	IdempotencyRepo repository.IdempotencyRepository
	ActiveProvider  string // default provider name used by the charge handler
	Addr            string
}

// NewServer wires up routes and returns a configured *http.Server.
func NewServer(deps Deps) *http.Server {
	mux := http.NewServeMux()

	chargeHandler := &handlers.ChargeHandler{
		Provider:        deps.Providers[deps.ActiveProvider],
		ChargeRepo:      deps.ChargeRepo,
		IdempotencyRepo: deps.IdempotencyRepo,
		ProviderID:      deps.ActiveProvider,
	}

	webhookHandler := &handlers.WebhookHandler{
		Providers:   deps.Providers,
		WebhookRepo: deps.WebhookRepo,
	}

	chargeStatusHandler := &handlers.ChargeStatusHandler{
		ChargeRepo: deps.ChargeRepo,
	}

	mux.HandleFunc("GET /health", handlers.Health)
	mux.Handle("POST /v1/charges", chargeHandler)
	mux.Handle("GET /v1/charges/{id}", chargeStatusHandler)
	mux.Handle("POST /v1/webhooks/{provider}", webhookHandler)

	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(apidocs.OpenAPISpec)
	})
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(apidocs.SwaggerUI)
	})

	handler := middleware.Recovery(middleware.Logging(mux))

	return &http.Server{
		Addr:         deps.Addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Run starts the server and blocks until SIGTERM or SIGINT, then shuts down gracefully.
func Run(srv *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
