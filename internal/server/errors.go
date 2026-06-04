package server

import (
	"net/http"

	"github.com/Tarasa24/psp-integration-demo/internal/server/response"
)

// WriteError writes a JSON error response. Delegates to response.WriteError.
func WriteError(w http.ResponseWriter, err error) {
	response.WriteError(w, err)
}
