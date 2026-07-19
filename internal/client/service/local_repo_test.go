package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-gophkeeper/internal/domain/models"
)

func tempDataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gophkeeper-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestLocalRepositoryCreateAndGet(t *testing.T) {
	repo, err := NewLocalRepository(tempDataDir(t))
	require.NoError(t, err)

	ctx := context.Background()
	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		Type:             models.SecretCredential,
		Title:            "Test Secret",
		Metadata:         []byte(`{"key":"value"}`),
		EncryptedPayload: []byte("encrypted-data"),
	}

	err = repo.Create(ctx, secret)
	require.NoError(t, err)

	// Получаем по ID.
	got, err := repo.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, secret.ID, got.ID)
	assert.Equal(t, secret.Title, got.Title)
	assert.Equal(t, secret.Type, got.Type)
	assert.Equal(t, secret.EncryptedPayload, got.EncryptedPayload)

	// Проверяем, что статус — pending_create.
	ls, err := repo.GetLocalByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, SyncStatusPendingCreate, ls.SyncStatus)
	assert.True(t, ls.LocalVers > 0)
}

func TestLocalRepositoryUpdate(t *testing.T) {
	repo, err := NewLocalRepository(tempDataDir(t))
	require.NoError(t, err)

	ctx := context.Background()
	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		Type:             models.SecretText,
		Title:            "Original",
		EncryptedPayload: []byte("original-data"),
	}

	err = repo.Create(ctx, secret)
	require.NoError(t, err)

	// Обновляем.
	secret.Title = "Updated"
	secret.EncryptedPayload = []byte("updated-data")
	err = repo.Update(ctx, secret)
	require.NoError(t, err)

	// Проверяем, что данные обновились.
	got, err := repo.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
	assert.Equal(t, []byte("updated-data"), got.EncryptedPayload)

	ls, err := repo.GetLocalByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, SyncStatusPendingCreate, ls.SyncStatus)
}

func TestLocalRepositoryDelete(t *testing.T) {
	repo, err := NewLocalRepository(tempDataDir(t))
	require.NoError(t, err)

	ctx := context.Background()
	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		Type:             models.SecretCard,
		Title:            "Delete Me",
		EncryptedPayload: []byte("data"),
	}

	err = repo.Create(ctx, secret)
	require.NoError(t, err)

	// Удаляем.
	err = repo.Delete(ctx, secret.ID, secret.UserID)
	require.NoError(t, err)

	// Должен быть недоступен через GetByID.
	_, err = repo.GetByID(ctx, secret.ID)
	assert.Error(t, err)

	// Но доступен через GetLocalByID со статусом pending_delete.
	ls, err := repo.GetLocalByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, SyncStatusPendingDelete, ls.SyncStatus)
	assert.NotNil(t, ls.DeletedAt)
}

func TestLocalRepositoryListByUser(t *testing.T) {
	repo, err := NewLocalRepository(tempDataDir(t))
	require.NoError(t, err)

	ctx := context.Background()
	userID := uuid.New()

	// Создаём 3 секрета для одного пользователя.
	for i := 0; i < 3; i++ {
		secret := &models.Secret{
			ID:               uuid.New(),
			UserID:           userID,
			Type:             models.SecretCredential,
			Title:            fmt.Sprintf("Secret %d", i),
			EncryptedPayload: []byte("data"),
		}
		err := repo.Create(ctx, secret)
		require.NoError(t, err)
	}

	// Создаём секрет для другого пользователя.
	otherSecret := &models.Secret{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		Type:             models.SecretText,
		Title:            "Other User Secret",
		EncryptedPayload: []byte("data"),
	}
	err = repo.Create(ctx, otherSecret)
	require.NoError(t, err)

	// ListByUser для первого пользователя — 3 секрета.
	metas, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, metas, 3)
}

func TestLocalRepositoryListByStatus(t *testing.T) {
	repo, err := NewLocalRepository(tempDataDir(t))
	require.NoError(t, err)

	ctx := context.Background()

	// Создаём 2 секрета.
	for i := 0; i < 2; i++ {
		secret := &models.Secret{
			ID:               uuid.New(),
			UserID:           uuid.New(),
			Type:             models.SecretCredential,
			Title:            fmt.Sprintf("Secret %d", i),
			EncryptedPayload: []byte("data"),
		}
		err := repo.Create(ctx, secret)
		require.NoError(t, err)
	}

	// Должны быть 2 pending_create.
	pending, err := repo.ListByStatus(ctx, SyncStatusPendingCreate)
	require.NoError(t, err)
	assert.Len(t, pending, 2)

	// synced — 0.
	synced, err := repo.ListByStatus(ctx, SyncStatusSynced)
	require.NoError(t, err)
	assert.Len(t, synced, 0)
}

func TestLocalRepositoryPullReplace(t *testing.T) {
	repo, err := NewLocalRepository(tempDataDir(t))
	require.NoError(t, err)

	ctx := context.Background()
	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		Type:             models.SecretBinary,
		Title:            "Pulled Secret",
		EncryptedPayload: []byte("pulled-data"),
	}

	err = repo.PullReplace(ctx, secret)
	require.NoError(t, err)

	// Проверяем статус — synced.
	ls, err := repo.GetLocalByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, SyncStatusSynced, ls.SyncStatus)
}

func TestLocalRepositoryPersistence(t *testing.T) {
	dir := tempDataDir(t)

	// Создаём репозиторий и добавляем секрет.
	repo1, err := NewLocalRepository(dir)
	require.NoError(t, err)

	ctx := context.Background()
	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		Type:             models.SecretCredential,
		Title:            "Persistent Secret",
		EncryptedPayload: []byte("persistent-data"),
	}

	err = repo1.Create(ctx, secret)
	require.NoError(t, err)

	// Проверяем, что файл кеша создан.
	cachePath := filepath.Join(dir, "cache.json")
	_, err = os.Stat(cachePath)
	require.NoError(t, err, "cache file should exist")
	require.FileExists(t, cachePath)

	// Создаём новый репозиторий из той же директории.
	repo2, err := NewLocalRepository(dir)
	require.NoError(t, err)

	// Проверяем, что секрет загрузился.
	got, err := repo2.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, secret.Title, got.Title)
	assert.Equal(t, secret.EncryptedPayload, got.EncryptedPayload)
}

func TestLocalRepositoryConcurrentAccess(t *testing.T) {
	repo, err := NewLocalRepository(tempDataDir(t))
	require.NoError(t, err)

	ctx := context.Background()
	userID := uuid.New()

	// Конкурентное создание 10 секретов.
	const count = 10
	done := make(chan struct{}, count)
	for i := 0; i < count; i++ {
		go func(i int) {
			secret := &models.Secret{
				ID:               uuid.New(),
				UserID:           userID,
				Type:             models.SecretCredential,
				Title:            fmt.Sprintf("Concurrent %d", i),
				EncryptedPayload: []byte("data"),
			}
			_ = repo.Create(ctx, secret)
			done <- struct{}{}
		}(i)
	}

	// Ждём завершения всех.
	for i := 0; i < count; i++ {
		<-done
	}

	// Проверяем, что все создались.
	metas, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, metas, count)
}
