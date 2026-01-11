package middleware

import (
	"context"
	"livon/internal/core/services"
	"net/http"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func AuthMiddleware(tokenSvc *services.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Bearer token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}
			// Validate Token
			phone, err := tokenSvc.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}
			// Inject UserID (phone) into Context
			ctx := context.WithValue(r.Context(), UserIDKey, phone)
			// Continue to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
