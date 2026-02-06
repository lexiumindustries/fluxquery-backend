package middleware

import (
	"log/slog"
	"net/http"
	"slices"
)

func CORS(allowedOrigins []string, env string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Determine allowed origin
			allow := false
			if slices.Contains(allowedOrigins, "*") {
				allow = true
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				if slices.Contains(allowedOrigins, origin) {
					allow = true
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}

			// Log in development
			if env == "development" {
				slog.Info("CORS Check", "origin", origin, "allowed", allow)
			}

			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Agent-Key, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Handle Preflight
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
