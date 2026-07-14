package interfaces

import "context"

// AuthService — контракт сервиса аутентификации.
type AuthService interface {
	// Register создаёт нового пользователя и возвращает JWT-токен.
	Register(ctx context.Context, email, password string) (token string, err error)

	// Login аутентифицирует пользователя и возвращает JWT-токен.
	Login(ctx context.Context, email, password string) (token string, err error)
}
