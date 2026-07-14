// Package interfaces определяет порты (контракты) для бизнес-логики приложения.
package interfaces

import (
	"context"

	"github.com/google/uuid"

	"go-gophkeeper/internal/domain/models"
)

// SecretData — незашифрованные данные секрета, которыми оперирует клиент.
// Хранятся только в памяти клиента, никогда не передаются на сервер в открытом виде.
type SecretData struct {
	// Type — тип хранимых данных (credential, text, card, binary).
	Type models.SecretType

	// Title — название секрета (отображается в списке, не шифруется).
	Title string

	// Metadata — произвольная метаинформация (не шифруется).
	// Используется для хранения сайта, банка или других заметок.
	Metadata map[string]string

	// Payload — сами данные секрета в структурированном виде.
	// Зависит от Type:
	//   credential: { "login": "...", "password": "..." }
	//   text:       { "content": "..." }
	//   card:       { "number": "...", "holder": "...", "cvv": "...", "expires": "..." }
	//   binary:     { "filename": "...", "content": "<base64>" }
	Payload map[string]string
}

// SecretListItem — элемент списка секретов (без payload).
type SecretListItem struct {
	ID        uuid.UUID
	Type      models.SecretType
	Title     string
	CreatedAt string
}

// SecretService — контракт сервиса хранения секретов на стороне клиента.
// Все данные шифруются перед отправкой на сервер (AES-256-GCM) и дешифруются после получения.
type SecretService interface {
	// Create создаёт новый секрет: шифрует Payload, отправляет на сервер.
	Create(ctx context.Context, data SecretData) (*models.Secret, error)

	// GetByID возвращает расшифрованный секрет по его ID.
	GetByID(ctx context.Context, id uuid.UUID) (*SecretData, error)

	// List возвращает список всех секретов пользователя (без payload).
	List(ctx context.Context) ([]SecretListItem, error)

	// Update обновляет существующий секрет: шифрует новый Payload, отправляет на сервер.
	Update(ctx context.Context, id uuid.UUID, data SecretData) error

	// Delete помечает секрет как удалённый.
	Delete(ctx context.Context, id uuid.UUID) error
}
