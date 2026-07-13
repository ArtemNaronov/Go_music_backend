package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/temic/go-music/internal/models"
)

// BearerToken validates the token from Authorization header or ?token= query param.
func BearerToken(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := tokenFromRequest(r)
			if token == "" || token != expected {
				writeUnauthorized(w, "Missing or invalid authorization token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func tokenFromRequest(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(models.APIError{
		Error:   true,
		Message: message,
	})
}
