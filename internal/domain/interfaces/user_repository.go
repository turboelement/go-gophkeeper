package interfaces

import (
	"context"

	"github.com/google/uuid"

	"go-gophkeeper/internal/domain/models"
)

// UserRepository — контракт доступа к данным пользователя.
type UserRepository interface {
	// Create сохраняет нового пользователя в хранилище.
	Create(ctx context.Context, user *models.User) error

	// GetByID возвращает пользователя по его идентификатору.
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	// GetByEmail возвращает пользователя по адресу электронной почты.
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	// Update обновляет данные существующего пользователя.
	Update(ctx context.Context, user *models.User) error

	// Delete удаляет пользователя по идентификатору.
	Delete(ctx context.Context, id uuid.UUID) error
}
