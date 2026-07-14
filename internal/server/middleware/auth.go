// Package middleware содержит HTTP-middleware для сервера.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"go-gophkeeper/internal/auth"
	domainerrors "go-gophkeeper/internal/domain/errors"
	"go-gophkeeper/internal/domain/interfaces"
)

// contextKey — неэкспортируемый тип для ключей контекста.
type contextKey string

const userIDKey contextKey = "user_id"

// GetUserID извлекает UserID из контекста.
// Возвращает nil, false если UserID не найден.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

// AuthMiddleware проверяет JWT-токен и сессию, кладёт UserID в контекст.
func AuthMiddleware(
	sessionRepo interfaces.SessionRepository,
	jwtSecret string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := extractBearerToken(r)
			if err != nil {
				http.Error(w, `{"error":"not authenticated"}`, http.StatusUnauthorized)
				return
			}

			claims, err := auth.Validate(tokenStr, jwtSecret)
			if err != nil {
				http.Error(w, `{"error":"not authenticated"}`, http.StatusUnauthorized)
				return
			}

			session, err := sessionRepo.GetByToken(r.Context(), tokenStr)
			if err != nil {
				if errors.Is(err, domainerrors.ErrNotFound) {
					http.Error(w, `{"error":"session expired"}`, http.StatusUnauthorized)
					return
				}
				http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
				return
			}
			if session == nil {
				http.Error(w, `{"error":"session expired"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid Authorization header format")
	}

	return parts[1], nil
}
