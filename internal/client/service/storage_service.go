// Package service содержит реализацию клиентской бизнес-логики GophKeeper.
// StorageService оперирует незашифрованными данными, шифрует их перед отправкой
// на сервер и дешифрует после получения (Zero Trust — сервер никогда не видит
// незашифрованные данные).
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"go-gophkeeper/internal/crypto"
	"go-gophkeeper/internal/domain/interfaces"
	"go-gophkeeper/internal/domain/models"
)

// SecretRepository — локальный alias для интерфейса репозитория секретов.
// Внедряется через DI, может быть HTTP/gRPC клиентом к серверу.
type SecretRepository = interfaces.SecretRepository

// StorageService — клиентский сервис для CRUD-операций с секретами.
// Все данные шифруются/дешифруются с использованием crypto.CryptoEngine
// перед отправкой на сервер и после получения.
//
// Поток данных:
//
//	Create: Payload → JSON → Encrypt(AES-256-GCM) → EncryptedPayload → Server
//	Get:    Server → EncryptedPayload → Decrypt(AES-256-GCM) → JSON → Payload
type StorageService struct {
	secretRepo interfaces.SecretRepository
	crypto     crypto.CryptoEngine
	userID     uuid.UUID
}

// NewStorageService создаёт StorageService.
//
// Параметры:
//   - secretRepo — репозиторий секретов (может быть реализацией HTTP/gRPC клиента)
//   - c — экземпляр CryptoEngine (AESGCMEngine с ключём, полученным через DeriveKey)
//   - userID — UUID текущего пользователя (из JWT)
func NewStorageService(secretRepo interfaces.SecretRepository, c crypto.CryptoEngine, userID uuid.UUID) *StorageService {
	return &StorageService{
		secretRepo: secretRepo,
		crypto:     c,
		userID:     userID,
	}
}

// Create создаёт новый секрет.
// Алгоритм:
//  1. Валидирует входные данные.
//  2. Сериализует Payload в JSON.
//  3. Шифрует JSON через crypto.Encrypt().
//  4. Отправляет зашифрованные данные через SecretRepository.Create().
func (s *StorageService) Create(ctx context.Context, data interfaces.SecretData) (*models.Secret, error) {
	if err := validateSecretData(data); err != nil {
		return nil, fmt.Errorf("storage: validate: %w", err)
	}

	// Сериализуем payload в JSON.
	payloadJSON, err := json.Marshal(data.Payload)
	if err != nil {
		return nil, fmt.Errorf("storage: marshal payload: %w", err)
	}

	// Шифруем.
	encryptedPayload, err := s.crypto.Encrypt(payloadJSON)
	if err != nil {
		return nil, fmt.Errorf("storage: encrypt: %w", err)
	}

	// Сериализуем metadata.
	var metadataJSON []byte
	if len(data.Metadata) > 0 {
		metadataJSON, err = json.Marshal(data.Metadata)
		if err != nil {
			return nil, fmt.Errorf("storage: marshal metadata: %w", err)
		}
	}

	now := time.Now()
	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           s.userID,
		Type:             data.Type,
		Title:            data.Title,
		Metadata:         metadataJSON,
		EncryptedPayload: encryptedPayload,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.secretRepo.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("storage: create: %w", err)
	}

	return secret, nil
}

// GetByID возвращает расшифрованный секрет по его ID.
// Алгоритм:
//  1. Запрашивает секрет через SecretRepository.GetByID().
//  2. Дешифрует EncryptedPayload через crypto.Decrypt().
//  3. Десериализует JSON обратно в Payload.
func (s *StorageService) GetByID(ctx context.Context, id uuid.UUID) (*interfaces.SecretData, error) {
	secret, err := s.secretRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("storage: get: %w", err)
	}

	// Проверяем владельца (дополнительная защита, хотя репозиторий тоже проверяет).
	if secret.UserID != s.userID {
		return nil, fmt.Errorf("storage: secret not found")
	}

	// Дешифруем.
	payloadJSON, err := s.crypto.Decrypt(secret.EncryptedPayload)
	if err != nil {
		return nil, fmt.Errorf("storage: decrypt: %w", err)
	}

	// Десериализуем payload.
	var payload map[string]string
	if len(payloadJSON) > 0 {
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			return nil, fmt.Errorf("storage: unmarshal payload: %w", err)
		}
	}

	// Десериализуем metadata.
	var metadata map[string]string
	if len(secret.Metadata) > 0 {
		if err := json.Unmarshal(secret.Metadata, &metadata); err != nil {
			// Метаданные могут быть в старом формате — нефатально.
			metadata = nil
		}
	}

	data := &interfaces.SecretData{
		Type:     secret.Type,
		Title:    secret.Title,
		Metadata: metadata,
		Payload:  payload,
	}

	return data, nil
}

// List возвращает список всех секретов пользователя (без зашифрованного payload).
// Список содержит только мета-информацию: ID, Type, Title, CreatedAt.
func (s *StorageService) List(ctx context.Context) ([]interfaces.SecretListItem, error) {
	metas, err := s.secretRepo.ListByUser(ctx, s.userID)
	if err != nil {
		return nil, fmt.Errorf("storage: list: %w", err)
	}

	items := make([]interfaces.SecretListItem, 0, len(metas))
	for _, m := range metas {
		items = append(items, interfaces.SecretListItem{
			ID:        m.ID,
			Type:      m.Type,
			Title:     m.Title,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		})
	}

	return items, nil
}

// Update обновляет существующий секрет.
// Новые данные шифруются перед отправкой на сервер.
func (s *StorageService) Update(ctx context.Context, id uuid.UUID, data interfaces.SecretData) error {
	if err := validateSecretData(data); err != nil {
		return fmt.Errorf("storage: validate: %w", err)
	}

	// Сериализуем payload в JSON.
	payloadJSON, err := json.Marshal(data.Payload)
	if err != nil {
		return fmt.Errorf("storage: marshal payload: %w", err)
	}

	// Шифруем.
	encryptedPayload, err := s.crypto.Encrypt(payloadJSON)
	if err != nil {
		return fmt.Errorf("storage: encrypt: %w", err)
	}

	// Сериализуем metadata.
	var metadataJSON []byte
	if len(data.Metadata) > 0 {
		metadataJSON, err = json.Marshal(data.Metadata)
		if err != nil {
			return fmt.Errorf("storage: marshal metadata: %w", err)
		}
	}

	secret := &models.Secret{
		ID:               id,
		UserID:           s.userID,
		Type:             data.Type,
		Title:            data.Title,
		Metadata:         metadataJSON,
		EncryptedPayload: encryptedPayload,
		UpdatedAt:        time.Now(),
	}

	if err := s.secretRepo.Update(ctx, secret); err != nil {
		return fmt.Errorf("storage: update: %w", err)
	}

	return nil
}

// Delete помечает секрет как удалённый (soft delete).
func (s *StorageService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.secretRepo.Delete(ctx, id, s.userID); err != nil {
		return fmt.Errorf("storage: delete: %w", err)
	}

	return nil
}

// validateSecretData проверяет корректность входных данных секрета.
func validateSecretData(data interfaces.SecretData) error {
	if data.Title == "" {
		return fmt.Errorf("title is required")
	}

	switch data.Type {
	case models.SecretCredential, models.SecretText, models.SecretCard, models.SecretBinary:
		// valid
	default:
		return fmt.Errorf("invalid secret type: %s", data.Type)
	}

	// Payload не должен быть пустым.
	if len(data.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}

	return nil
}

// SetUserID обновляет UserID в StorageService (например, после логина).
func (s *StorageService) SetUserID(userID uuid.UUID) {
	s.userID = userID
}

// Ensure interfaces compliance.
var _ interfaces.SecretService = (*StorageService)(nil)
