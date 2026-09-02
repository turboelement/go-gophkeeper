package interfaces

import (
	"context"
	"time"

	"go-gophkeeper/internal/domain/models"
)

// SessionRepository — контракт доступа к данным сессий.
type SessionRepository interface {
	// Create сохраняет новую сессию в хранилище.
	Create(ctx context.Context, session *models.Session) error

	// GetByToken возвращает сессию по её токену.
	GetByToken(ctx context.Context, token string) (*models.Session, error)

	// Delete удаляет сессию по токену.
	Delete(ctx context.Context, token string) error

	// CleanupExpired удаляет все сессии, истёкшие до указанного времени.
	CleanupExpired(ctx context.Context, before time.Time) error
}
