package models

import (
	"time"

	"github.com/google/uuid"
)

// Session — аутентифицированная сессия пользователя.
type Session struct {
	Token     string
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}
