package interfaces

import (
	"context"

	"github.com/google/uuid"

	"go-gophkeeper/internal/domain/models"
)

// SecretRepository — контракт доступа к данным секретов.
type SecretRepository interface {
	// Create сохраняет новый секрет в хранилище.
	Create(ctx context.Context, secret *models.Secret) error

	// GetByID возвращает секрет по его идентификатору.
	GetByID(ctx context.Context, id uuid.UUID) (*models.Secret, error)

	// ListByUser возвращает мета-информацию о всех неудалённых секретах пользователя.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.SecretMeta, error)

	// Update обновляет данные существующего секрета.
	Update(ctx context.Context, secret *models.Secret) error

	// Delete помечает секрет как удалённый (soft delete) по идентификатору в рамках пользователя.
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
