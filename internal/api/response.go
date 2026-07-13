package api

import (
	"encoding/json"
	"net/http"

	"github.com/temic/go-music/internal/models"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.APIError{
		Error:   true,
		Message: message,
	})
}
