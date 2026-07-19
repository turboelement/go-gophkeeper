package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"go-gophkeeper/internal/domain/models"
)

// SyncStatus — статус синхронизации секрета с сервером.
type SyncStatus string

const (
	SyncStatusSynced        SyncStatus = "synced"
	SyncStatusPendingCreate SyncStatus = "pending_create"
	SyncStatusPendingUpdate SyncStatus = "pending_update"
	SyncStatusPendingDelete SyncStatus = "pending_delete"
	SyncStatusConflict      SyncStatus = "conflict"
)

// LocalSecret — секрет с метаданными синхронизации для локального кеша.
type LocalSecret struct {
	*models.Secret
	SyncStatus SyncStatus `json:"sync_status"`
	LocalVers  int64      `json:"local_vers"`
}

// cacheFile — структура для сериализации всего кеша в JSON.
type cacheFile struct {
	Secrets  []*LocalSecret `json:"secrets"`
	NextVers int64          `json:"next_vers"`
}

// LocalRepository — локальное in-memory хранилище секретов с JSON-персистентностью.
// Реализует интерфейс interfaces.SecretRepository.
// Все данные хранятся в зашифрованном виде (EncryptedPayload), ключ шифрования —
// только в памяти клиента.
type LocalRepository struct {
	mu       sync.RWMutex
	secrets  map[uuid.UUID]*LocalSecret
	filePath string
	nextVers int64
}

// NewLocalRepository создаёт LocalRepository и загружает кеш из файла, если он существует.
func NewLocalRepository(dataDir string) (*LocalRepository, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("local: create data dir: %w", err)
	}

	filePath := filepath.Join(dataDir, "cache.json")
	repo := &LocalRepository{
		secrets:  make(map[uuid.UUID]*LocalSecret),
		filePath: filePath,
		nextVers: 1,
	}

	// Загружаем кеш из файла, если существует.
	if err := repo.loadCache(); err != nil {
		return nil, fmt.Errorf("local: load cache: %w", err)
	}

	return repo, nil
}

// Create сохраняет секрет локально со статусом SyncStatusPendingCreate.
func (r *LocalRepository) Create(_ context.Context, secret *models.Secret) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.secrets[secret.ID]; exists {
		return fmt.Errorf("local: secret already exists")
	}

	localSecret := &LocalSecret{
		Secret:     copySecret(secret),
		SyncStatus: SyncStatusPendingCreate,
		LocalVers:  r.nextVers,
	}
	r.nextVers++
	r.secrets[secret.ID] = localSecret

	return r.saveCache()
}

// GetByID возвращает копию секрета по ID.
func (r *LocalRepository) GetByID(_ context.Context, id uuid.UUID) (*models.Secret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ls, exists := r.secrets[id]
	if !exists || ls.DeletedAt != nil {
		return nil, fmt.Errorf("local: secret not found")
	}

	return copySecret(ls.Secret), nil
}

// ListByUser возвращает мета-информацию о всех неудалённых секретах пользователя.
func (r *LocalRepository) ListByUser(_ context.Context, userID uuid.UUID) ([]models.SecretMeta, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var metas []models.SecretMeta
	for _, ls := range r.secrets {
		if ls.UserID == userID && ls.DeletedAt == nil {
			metas = append(metas, models.SecretMeta{
				ID:        ls.ID,
				Type:      ls.Type,
				Title:     ls.Title,
				CreatedAt: ls.CreatedAt,
			})
		}
	}

	if metas == nil {
		metas = []models.SecretMeta{}
	}

	return metas, nil
}

// ListByStatus возвращает все секреты с указанным статусом синхронизации.
func (r *LocalRepository) ListByStatus(_ context.Context, status SyncStatus) ([]*LocalSecret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*LocalSecret
	for _, ls := range r.secrets {
		if ls.SyncStatus == status {
			cp := &LocalSecret{
				Secret:     copySecret(ls.Secret),
				SyncStatus: ls.SyncStatus,
				LocalVers:  ls.LocalVers,
			}
			result = append(result, cp)
		}
	}

	return result, nil
}

// GetLocalByID возвращает LocalSecret (со статусом синхронизации) по ID.
// Используется SyncService для проверки статуса перед pull/push.
func (r *LocalRepository) GetLocalByID(_ context.Context, id uuid.UUID) (*LocalSecret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ls, exists := r.secrets[id]
	if !exists {
		return nil, fmt.Errorf("local: secret not found")
	}

	cp := &LocalSecret{
		Secret:     copySecret(ls.Secret),
		SyncStatus: ls.SyncStatus,
		LocalVers:  ls.LocalVers,
	}

	return cp, nil
}

// Update обновляет локальный секрет и меняет статус на PendingUpdate (если был Synced).
func (r *LocalRepository) Update(_ context.Context, secret *models.Secret) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ls, exists := r.secrets[secret.ID]
	if !exists {
		return fmt.Errorf("local: secret not found")
	}

	// Сохраняем старый статус, обновляем данные.
	oldStatus := ls.SyncStatus
	ls.Secret = copySecret(secret)
	ls.LocalVers = r.nextVers
	r.nextVers++

	// Если секрет был synced или conflict — меняем на pending_update.
	// Если уже pending_create/pending_update/pending_delete — оставляем.
	switch oldStatus {
	case SyncStatusSynced:
		ls.SyncStatus = SyncStatusPendingUpdate
	case SyncStatusConflict:
		ls.SyncStatus = SyncStatusPendingUpdate
		// Для pending_create/pending_update/pending_delete — остаётся как есть.
	}

	return r.saveCache()
}

// Delete помечает секрет как удалённый (soft delete), статус — PendingDelete.
func (r *LocalRepository) Delete(_ context.Context, id uuid.UUID, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ls, exists := r.secrets[id]
	if !exists || ls.UserID != userID {
		return fmt.Errorf("local: secret not found")
	}

	now := time.Now()
	ls.DeletedAt = &now
	ls.SyncStatus = SyncStatusPendingDelete
	ls.LocalVers = r.nextVers
	r.nextVers++

	return r.saveCache()
}

// SetStatus изменяет статус синхронизации секрета.
func (r *LocalRepository) SetStatus(_ context.Context, id uuid.UUID, status SyncStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ls, exists := r.secrets[id]
	if !exists {
		return fmt.Errorf("local: secret not found")
	}

	ls.SyncStatus = status
	return r.saveCache()
}

// Remove удаляет секрет из локального кеша полностью (после подтверждения сервером).
func (r *LocalRepository) Remove(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.secrets[id]; !exists {
		return fmt.Errorf("local: secret not found")
	}

	delete(r.secrets, id)
	return r.saveCache()
}

// PullReplace заменяет/добавляет секрет из серверной версии (при Pull).
// Статус устанавливается в synced, локальный vers увеличивается.
func (r *LocalRepository) PullReplace(_ context.Context, secret *models.Secret) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ls := &LocalSecret{
		Secret:     copySecret(secret),
		SyncStatus: SyncStatusSynced,
		LocalVers:  r.nextVers,
	}
	r.nextVers++
	r.secrets[secret.ID] = ls

	return r.saveCache()
}

// saveCache сериализует кеш в JSON и пишет в файл.
func (r *LocalRepository) saveCache() error {
	secrets := make([]*LocalSecret, 0, len(r.secrets))
	for _, ls := range r.secrets {
		secrets = append(secrets, ls)
	}

	data, err := json.MarshalIndent(cacheFile{
		Secrets:  secrets,
		NextVers: r.nextVers,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("local: marshal cache: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0600); err != nil {
		return fmt.Errorf("local: write cache: %w", err)
	}

	return nil
}

// loadCache загружает кеш из JSON-файла.
func (r *LocalRepository) loadCache() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Нет файла — не ошибка, начинаем с пустым кешем.
		}
		return fmt.Errorf("local: read cache: %w", err)
	}

	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return fmt.Errorf("local: unmarshal cache: %w", err)
	}

	r.secrets = make(map[uuid.UUID]*LocalSecret, len(cf.Secrets))
	for _, ls := range cf.Secrets {
		if ls != nil && ls.Secret != nil {
			r.secrets[ls.ID] = ls
		}
	}
	if cf.NextVers > 0 {
		r.nextVers = cf.NextVers
	}

	return nil
}

// copySecret создаёт глубокую копию модели Secret.
func copySecret(s *models.Secret) *models.Secret {
	if s == nil {
		return nil
	}

	cp := &models.Secret{
		ID:        s.ID,
		UserID:    s.UserID,
		Type:      s.Type,
		Title:     s.Title,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}

	if s.DeletedAt != nil {
		d := *s.DeletedAt
		cp.DeletedAt = &d
	}

	if len(s.Metadata) > 0 {
		cp.Metadata = make([]byte, len(s.Metadata))
		copy(cp.Metadata, s.Metadata)
	}

	if len(s.EncryptedPayload) > 0 {
		cp.EncryptedPayload = make([]byte, len(s.EncryptedPayload))
		copy(cp.EncryptedPayload, s.EncryptedPayload)
	}

	return cp
}
