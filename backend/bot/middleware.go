package main

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// authMiddleware validates Bearer token authentication
func authMiddleware(apiToken string, logger zerolog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			logger.Warn().Str("path", r.URL.Path).Str("method", r.Method).Str("remote", r.RemoteAddr).Msg("unauthorized: missing authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			logger.Warn().Str("path", r.URL.Path).Str("method", r.Method).Str("remote", r.RemoteAddr).Msg("unauthorized: invalid authorization header format")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if subtle.ConstantTimeCompare([]byte(bearerToken[1]), []byte(apiToken)) != 1 {
			logger.Warn().Str("path", r.URL.Path).Str("method", r.Method).Str("remote", r.RemoteAddr).Msg("unauthorized: invalid token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		logger.Debug().Str("path", r.URL.Path).Str("method", r.Method).Msg("authorized request")
		next.ServeHTTP(w, r)
	})
}
