package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-gophkeeper/internal/crypto"
	"go-gophkeeper/internal/domain/interfaces"
	"go-gophkeeper/internal/domain/models"
)

// mockSecretRepo — простая in-memory реализация SecretRepository для тестов.
type mockSecretRepo struct {
	secrets map[uuid.UUID]*models.Secret
}

func newMockSecretRepo() *mockSecretRepo {
	return &mockSecretRepo{
		secrets: make(map[uuid.UUID]*models.Secret),
	}
}

func (m *mockSecretRepo) Create(_ context.Context, secret *models.Secret) error {
	secretCopy := *secret
	if secret.Metadata != nil {
		metaCopy := make([]byte, len(secret.Metadata))
		copy(metaCopy, secret.Metadata)
		secretCopy.Metadata = metaCopy
	}
	payloadCopy := make([]byte, len(secret.EncryptedPayload))
	copy(payloadCopy, secret.EncryptedPayload)
	secretCopy.EncryptedPayload = payloadCopy

	if _, exists := m.secrets[secret.ID]; exists {
		return assert.AnError
	}
	m.secrets[secret.ID] = &secretCopy
	return nil
}

func (m *mockSecretRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Secret, error) {
	s, exists := m.secrets[id]
	if !exists {
		return nil, assert.AnError
	}
	sCopy := *s
	payloadCopy := make([]byte, len(s.EncryptedPayload))
	copy(payloadCopy, s.EncryptedPayload)
	sCopy.EncryptedPayload = payloadCopy
	return &sCopy, nil
}

func (m *mockSecretRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]models.SecretMeta, error) {
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

func (m *mockSecretRepo) Update(_ context.Context, secret *models.Secret) error {
	if _, exists := m.secrets[secret.ID]; !exists {
		return assert.AnError
	}
	sCopy := *secret
	payloadCopy := make([]byte, len(secret.EncryptedPayload))
	copy(payloadCopy, secret.EncryptedPayload)
	sCopy.EncryptedPayload = payloadCopy
	m.secrets[secret.ID] = &sCopy
	return nil
}

func (m *mockSecretRepo) Delete(_ context.Context, id uuid.UUID, userID uuid.UUID) error {
	s, exists := m.secrets[id]
	if !exists || s.UserID != userID {
		return assert.AnError
	}
	now := time.Now()
	s.DeletedAt = &now
	return nil
}

// setupTestService создаёт StorageService с тестовым ключом и mock-репозиторием.
func setupTestService(t *testing.T) (*StorageService, uuid.UUID) {
	t.Helper()

	userID := uuid.New()

	// Создаём детерминированный ключ для тестов (быстрый Argon2 с минимальными параметрами).
	key, err := crypto.DeriveKeyWithParams("test-password", userID, 1, 64*1024, 1)
	require.NoError(t, err)

	engine := crypto.NewAESGCMEngine(key)
	repo := newMockSecretRepo()

	svc := NewStorageService(repo, engine, userID)
	return svc, userID
}

func TestStorageService_CreateAndGet(t *testing.T) {
	t.Parallel()

	svc, _ := setupTestService(t)
	ctx := context.Background()

	t.Run("credential", func(t *testing.T) {
		data := interfaces.SecretData{
			Type:  models.SecretCredential,
			Title: "My GitHub Account",
			Metadata: map[string]string{
				"website": "https://github.com",
				"note":    "personal account",
			},
			Payload: map[string]string{
				"login":    "john.doe",
				"password": "s3cret!pass",
			},
		}

		secret, err := svc.Create(ctx, data)
		require.NoError(t, err)
		assert.Equal(t, models.SecretCredential, secret.Type)
		assert.Equal(t, "My GitHub Account", secret.Title)
		assert.NotEmpty(t, secret.EncryptedPayload, "payload should be encrypted")

		// Проверяем, что в репозитории сохранён шифротекст, а не plaintext.
		require.NotNil(t, secret.EncryptedPayload)
		assert.NotContains(t, string(secret.EncryptedPayload), "john.doe",
			"plaintext should not appear in EncryptedPayload")

		// Получаем и дешифруем.
		gotData, err := svc.GetByID(ctx, secret.ID)
		require.NoError(t, err)
		assert.Equal(t, data.Type, gotData.Type)
		assert.Equal(t, data.Title, gotData.Title)
		assert.Equal(t, data.Payload, gotData.Payload)
		assert.Equal(t, data.Metadata, gotData.Metadata)
	})

	t.Run("text", func(t *testing.T) {
		data := interfaces.SecretData{
			Type:  models.SecretText,
			Title: "Secret Note",
			Payload: map[string]string{
				"content": "This is a very secret note with unicode: привет! 🚀",
			},
		}

		secret, err := svc.Create(ctx, data)
		require.NoError(t, err)

		gotData, err := svc.GetByID(ctx, secret.ID)
		require.NoError(t, err)
		assert.Equal(t, data.Payload, gotData.Payload)
	})

	t.Run("card", func(t *testing.T) {
		data := interfaces.SecretData{
			Type:  models.SecretCard,
			Title: "My Visa Card",
			Payload: map[string]string{
				"number":  "4111111111111111",
				"holder":  "JOHN DOE",
				"cvv":     "123",
				"expires": "12/28",
			},
		}

		secret, err := svc.Create(ctx, data)
		require.NoError(t, err)

		gotData, err := svc.GetByID(ctx, secret.ID)
		require.NoError(t, err)
		assert.Equal(t, data.Payload, gotData.Payload)
	})

	t.Run("binary", func(t *testing.T) {
		data := interfaces.SecretData{
			Type:  models.SecretBinary,
			Title: "ssh key",
			Payload: map[string]string{
				"filename": "id_rsa",
				"content":  "-----BEGIN OPENSSH PRIVATE KEY-----\nbase64encoded...\n-----END OPENSSH PRIVATE KEY-----",
			},
		}

		secret, err := svc.Create(ctx, data)
		require.NoError(t, err)

		gotData, err := svc.GetByID(ctx, secret.ID)
		require.NoError(t, err)
		assert.Equal(t, data.Payload, gotData.Payload)
	})
}

func TestStorageService_Create_Validation(t *testing.T) {
	t.Parallel()

	svc, _ := setupTestService(t)
	ctx := context.Background()

	tests := []struct {
		name string
		data interfaces.SecretData
	}{
		{
			name: "empty title",
			data: interfaces.SecretData{
				Type:    models.SecretText,
				Title:   "",
				Payload: map[string]string{"content": "test"},
			},
		},
		{
			name: "invalid type",
			data: interfaces.SecretData{
				Type:    "unknown",
				Title:   "test",
				Payload: map[string]string{"content": "test"},
			},
		},
		{
			name: "empty payload",
			data: interfaces.SecretData{
				Type:    models.SecretText,
				Title:   "test",
				Payload: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.Create(ctx, tt.data)
			assert.Error(t, err)
		})
	}
}

func TestStorageService_List(t *testing.T) {
	t.Parallel()

	svc, _ := setupTestService(t)
	ctx := context.Background()

	// Создаём несколько секретов.
	creds := []interfaces.SecretData{
		{
			Type:    models.SecretCredential,
			Title:   "GitHub",
			Payload: map[string]string{"login": "user1", "password": "pass1"},
		},
		{
			Type:    models.SecretCredential,
			Title:   "GitLab",
			Payload: map[string]string{"login": "user2", "password": "pass2"},
		},
		{
			Type:    models.SecretCard,
			Title:   "My Card",
			Payload: map[string]string{"number": "4111", "holder": "ME"},
		},
	}

	for _, d := range creds {
		_, err := svc.Create(ctx, d)
		require.NoError(t, err)
	}

	items, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, items, 3)

	// Проверяем, что список содержит только мета-информацию.
	for _, item := range items {
		assert.NotEmpty(t, item.ID)
		assert.NotEmpty(t, item.Title)
		assert.NotEmpty(t, item.CreatedAt)
	}
}

func TestStorageService_Update(t *testing.T) {
	t.Parallel()

	svc, _ := setupTestService(t)
	ctx := context.Background()

	// Создаём.
	original := interfaces.SecretData{
		Type:    models.SecretCredential,
		Title:   "Old Title",
		Payload: map[string]string{"login": "old", "password": "oldpass"},
	}
	secret, err := svc.Create(ctx, original)
	require.NoError(t, err)

	// Обновляем.
	updated := interfaces.SecretData{
		Type:    models.SecretCredential,
		Title:   "Updated Title",
		Payload: map[string]string{"login": "new", "password": "newpass"},
		Metadata: map[string]string{
			"website": "https://example.com",
		},
	}
	err = svc.Update(ctx, secret.ID, updated)
	require.NoError(t, err)

	// Проверяем.
	gotData, err := svc.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", gotData.Title)
	assert.Equal(t, "new", gotData.Payload["login"])
	assert.Equal(t, "newpass", gotData.Payload["password"])
	assert.Equal(t, "https://example.com", gotData.Metadata["website"])
}

func TestStorageService_Delete(t *testing.T) {
	t.Parallel()

	svc, _ := setupTestService(t)
	ctx := context.Background()

	data := interfaces.SecretData{
		Type:    models.SecretText,
		Title:   "To Delete",
		Payload: map[string]string{"content": "secret"},
	}
	secret, err := svc.Create(ctx, data)
	require.NoError(t, err)

	// Удаляем.
	err = svc.Delete(ctx, secret.ID)
	require.NoError(t, err)

	// Проверяем, что секрет не возвращается в списке.
	items, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestStorageService_DifferentUsersCantReadEachOther(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Создаём два сервиса для разных пользователей (разные ключи шифрования).
	svc1, userID1 := setupTestService(t)
	svc2, userID2 := setupTestService(t)
	assert.NotEqual(t, userID1, userID2)

	// Пользователь 1 создаёт секрет.
	data := interfaces.SecretData{
		Type:    models.SecretText,
		Title:   "User1 Secret",
		Payload: map[string]string{"content": "only user1 can read this"},
	}
	secret, err := svc1.Create(ctx, data)
	require.NoError(t, err)

	// Пользователь 2 пытается прочитать — не должен найти (ID чужой).
	_, err = svc2.GetByID(ctx, secret.ID)
	assert.Error(t, err, "user2 should not be able to read user1's secret")

	// Пользователь 2 не видит секрет в списке.
	items, err := svc2.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, items, "user2 should see no secrets")
}

func TestStorageService_EncryptedPayloadIntegrity(t *testing.T) {
	t.Parallel()

	svc, _ := setupTestService(t)
	ctx := context.Background()

	// Создаём секрет.
	data := interfaces.SecretData{
		Type:    models.SecretText,
		Title:   "Integrity Test",
		Payload: map[string]string{"content": "sensitive data"},
	}
	secret, err := svc.Create(ctx, data)
	require.NoError(t, err)

	// Повреждаем шифротекст напрямую в репозитории (симуляция атаки на БД).
	repo := svc.secretRepo.(*mockSecretRepo)
	corrupted := repo.secrets[secret.ID]
	corrupted.EncryptedPayload[len(corrupted.EncryptedPayload)/2] ^= 0xFF

	// Попытка дешифровать должна провалиться (GCM обнаружит подмену).
	_, err = svc.GetByID(ctx, secret.ID)
	assert.Error(t, err, "tampered ciphertext should be detected by GCM")
}

func TestStorageService_ListByType(t *testing.T) {
	t.Parallel()

	svc, _ := setupTestService(t)
	ctx := context.Background()

	// Создаём секреты разных типов.
	secrets := []interfaces.SecretData{
		{Type: models.SecretCredential, Title: "Login1", Payload: map[string]string{"login": "a", "password": "b"}},
		{Type: models.SecretCard, Title: "Card1", Payload: map[string]string{"number": "1", "holder": "Me"}},
		{Type: models.SecretText, Title: "Note1", Payload: map[string]string{"content": "c"}},
		{Type: models.SecretBinary, Title: "File1", Payload: map[string]string{"filename": "f", "content": "d"}},
	}

	for _, s := range secrets {
		_, err := svc.Create(ctx, s)
		require.NoError(t, err)
	}

	items, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, items, 4)
}

func TestStorageService_LargePayload(t *testing.T) {
	t.Parallel()

	svc, _ := setupTestService(t)
	ctx := context.Background()

	// Создаём большой payload (~50KB).
	largeContent := make([]byte, 50*1024)
	for i := range largeContent {
		largeContent[i] = 'A' + byte(i%26)
	}

	data := interfaces.SecretData{
		Type:    models.SecretBinary,
		Title:   "Large File",
		Payload: map[string]string{"filename": "large.bin", "content": string(largeContent)},
	}

	secret, err := svc.Create(ctx, data)
	require.NoError(t, err)

	gotData, err := svc.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, data.Payload, gotData.Payload)
	assert.Len(t, gotData.Payload["content"], 50*1024)
}

// TestStorageService_NoPlaintextInLogs — проверяет, что в зашифрованных данных
// нет следов plaintext (косвенная проверка, что шифрование работает).
func TestStorageService_NoPlaintextInLogs(t *testing.T) {
	t.Parallel()

	svc, _ := setupTestService(t)
	ctx := context.Background()

	secretWord := "ultra-secret-password-12345"
	data := interfaces.SecretData{
		Type:    models.SecretCredential,
		Title:   "Secret Test",
		Payload: map[string]string{"login": "admin", "password": secretWord},
	}

	secret, err := svc.Create(ctx, data)
	require.NoError(t, err)

	// Проверяем, что secretWord не встречается в EncryptedPayload.
	assert.NotContains(t, string(secret.EncryptedPayload), secretWord)

	// Проверяем сырой JSON в репозитории.
	rawJSON, err := json.Marshal(secret)
	require.NoError(t, err)
	assert.NotContains(t, string(rawJSON), secretWord)
}
