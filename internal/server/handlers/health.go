package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Tarasa24/psp-integration-demo/internal/server/response"
)

// Health handles GET /health.
func Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response.HealthResponse{Status: "ok"})
}
