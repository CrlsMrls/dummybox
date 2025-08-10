package server

import (
	"net/http"

	"github.com/crlsmrls/dummybox/config"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// CorrelationIDMiddleware adds a correlation ID to the request context and response headers.
func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		w.Header().Set("X-Correlation-ID", correlationID)

		log := hlog.FromRequest(r)
		log.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("correlation_id", correlationID)
		})

		next.ServeHTTP(w, r)
	})
}

// TokenAuthMiddleware provides simple token-based authentication for command endpoints.
// It checks for token in GET parameter "token" or in "X-Auth-Token" header.
func TokenAuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no auth token is configured, allow access
			if cfg.AuthToken == "" {
				next.ServeHTTP(w, r)
				return
			}

			var providedToken string

			// Check for token in query parameter first
			if tokenParam := r.URL.Query().Get("token"); tokenParam != "" {
				providedToken = tokenParam
			} else if authHeader := r.Header.Get("X-Auth-Token"); authHeader != "" {
				providedToken = authHeader
			}

			// If no token provided, return unauthorized
			if providedToken == "" {
				log := hlog.FromRequest(r)
				log.Warn().Msg("missing authentication token for protected endpoint")
				http.Error(w, "Unauthorized: token required", http.StatusUnauthorized)
				return
			}

			// Compare with the configured auth token
			if providedToken != cfg.AuthToken {
				log := hlog.FromRequest(r)
				log.Warn().Msg("invalid authentication token for protected endpoint")
				http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
				return
			}

			// Authentication successful
			log := hlog.FromRequest(r)
			log.Info().Msg("successful token authentication for protected endpoint")
			next.ServeHTTP(w, r)
		})
	}
}
