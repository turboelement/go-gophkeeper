package interfaces

import (
	"context"

	"github.com/google/uuid"
)

// AuthService — контракт сервиса аутентификации.
type AuthService interface {
	// Register создаёт нового пользователя и возвращает JWT-токен и UUID пользователя.
	Register(ctx context.Context, email, password string) (token string, userID uuid.UUID, err error)

	// Login аутентифицирует пользователя и возвращает JWT-токен и UUID пользователя.
	Login(ctx context.Context, email, password string) (token string, userID uuid.UUID, err error)
}
