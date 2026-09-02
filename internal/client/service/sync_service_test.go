package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go-gophkeeper/internal/crypto"
	"go-gophkeeper/internal/domain/models"
)

// mockRemoteRepo реализует interfaces.SecretRepository для тестов SyncService.
type mockRemoteRepo struct {
	mu      sync.RWMutex
	secrets map[uuid.UUID]*models.Secret
}

func newMockRemoteRepo() *mockRemoteRepo {
	return &mockRemoteRepo{
		secrets: make(map[uuid.UUID]*models.Secret),
	}
}

func (m *mockRemoteRepo) Create(_ context.Context, secret *models.Secret) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.secrets[secret.ID]; exists {
		return fmt.Errorf("already exists")
	}
	cp := *secret
	cp.CreatedAt = time.Now()
	cp.UpdatedAt = time.Now()
	m.secrets[secret.ID] = &cp
	return nil
}

func (m *mockRemoteRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Secret, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, exists := m.secrets[id]
	if !exists || s.DeletedAt != nil {
		return nil, fmt.Errorf("not found")
	}
	cp := *s
	return &cp, nil
}

func (m *mockRemoteRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]models.SecretMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var metas []models.SecretMeta
	for _, s := range m.secrets {
		if s.UserID == userID && s.DeletedAt == nil {
			metas = append(metas, models.SecretMeta{
				ID:        s.ID,
				Type:      s.Type,
				Title:     s.Title,
				CreatedAt: s.CreatedAt,
			})
		}
	}
	return metas, nil
}

func (m *mockRemoteRepo) Update(_ context.Context, secret *models.Secret) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, exists := m.secrets[secret.ID]
	if !exists {
		return fmt.Errorf("not found")
	}
	s.Title = secret.Title
	s.EncryptedPayload = secret.EncryptedPayload
	s.UpdatedAt = time.Now()
	return nil
}

func (m *mockRemoteRepo) Delete(_ context.Context, id uuid.UUID, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, exists := m.secrets[id]
	if !exists {
		return fmt.Errorf("not found")
	}
	now := time.Now()
	s.DeletedAt = &now
	return nil
}

func newSyncTestEnv(t *testing.T) (*LocalRepository, *mockRemoteRepo, *SyncService, uuid.UUID) {
	t.Helper()

	dir, err := os.MkdirTemp("", "gophkeeper-sync-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	localRepo, err := NewLocalRepository(dir)
	require.NoError(t, err)

	mockRemote := newMockRemoteRepo()

	userID := uuid.New()

	var key [crypto.KeySize]byte
	for i := range key {
		key[i] = byte(i)
	}
	engine := crypto.NewAESGCMEngine(key)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	syncSvc := NewSyncService(localRepo, mockRemote, engine, userID, logger)

	return localRepo, mockRemote, syncSvc, userID
}

func TestSyncPushCreate(t *testing.T) {
	localRepo, mockRemote, syncSvc, userID := newSyncTestEnv(t)
	ctx := context.Background()

	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           userID,
		Type:             models.SecretCredential,
		Title:            "Push Test",
		EncryptedPayload: []byte("encrypted"),
	}
	err := localRepo.Create(ctx, secret)
	require.NoError(t, err)

	count, err := syncSvc.Push(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	remoteSecret, err := mockRemote.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, secret.Title, remoteSecret.Title)

	ls, err := localRepo.GetLocalByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, SyncStatusSynced, ls.SyncStatus)
}

func TestSyncPushDelete(t *testing.T) {
	localRepo, mockRemote, syncSvc, userID := newSyncTestEnv(t)
	ctx := context.Background()

	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           userID,
		Type:             models.SecretCredential,
		Title:            "Delete Test",
		EncryptedPayload: []byte("encrypted"),
	}
	err := mockRemote.Create(ctx, secret)
	require.NoError(t, err)

	err = localRepo.PullReplace(ctx, secret)
	require.NoError(t, err)
	err = localRepo.SetStatus(ctx, secret.ID, SyncStatusPendingDelete)
	require.NoError(t, err)

	count, err := syncSvc.Push(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	_, err = mockRemote.GetByID(ctx, secret.ID)
	require.Error(t, err)

	_, err = localRepo.GetLocalByID(ctx, secret.ID)
	require.Error(t, err)
}

func TestSyncPushUpdate(t *testing.T) {
	localRepo, mockRemote, syncSvc, userID := newSyncTestEnv(t)
	ctx := context.Background()

	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           userID,
		Type:             models.SecretText,
		Title:            "Original",
		EncryptedPayload: []byte("original"),
	}
	err := mockRemote.Create(ctx, secret)
	require.NoError(t, err)

	secret.Title = "Updated"
	secret.EncryptedPayload = []byte("updated")
	err = localRepo.PullReplace(ctx, secret)
	require.NoError(t, err)
	err = localRepo.SetStatus(ctx, secret.ID, SyncStatusPendingUpdate)
	require.NoError(t, err)

	count, err := syncSvc.Push(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	remoteSecret, err := mockRemote.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", remoteSecret.Title)
	assert.Equal(t, []byte("updated"), remoteSecret.EncryptedPayload)
}

func TestSyncPullNewSecret(t *testing.T) {
	localRepo, mockRemote, syncSvc, userID := newSyncTestEnv(t)
	ctx := context.Background()

	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           userID,
		Type:             models.SecretCard,
		Title:            "Pulled Card",
		EncryptedPayload: []byte("card-data"),
	}
	err := mockRemote.Create(ctx, secret)
	require.NoError(t, err)

	count, err := syncSvc.Pull(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	localSecret, err := localRepo.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, secret.Title, localSecret.Title)

	ls, err := localRepo.GetLocalByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, SyncStatusSynced, ls.SyncStatus)
}

func TestSyncPullConflict(t *testing.T) {
	localRepo, mockRemote, syncSvc, userID := newSyncTestEnv(t)
	ctx := context.Background()

	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           userID,
		Type:             models.SecretText,
		Title:            "Server Version",
		EncryptedPayload: []byte("server-data"),
	}
	err := mockRemote.Create(ctx, secret)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	localSecret := &models.Secret{
		ID:               secret.ID,
		UserID:           userID,
		Type:             models.SecretText,
		Title:            "Local Version",
		EncryptedPayload: []byte("local-data"),
	}
	err = localRepo.PullReplace(ctx, localSecret)
	require.NoError(t, err)

	count, err := syncSvc.Pull(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	updated, err := localRepo.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, "Server Version", updated.Title)
	assert.Equal(t, []byte("server-data"), updated.EncryptedPayload)
}
