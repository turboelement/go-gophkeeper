package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "go-gophkeeper/internal/domain/errors"
	"go-gophkeeper/internal/domain/models"
)

func TestNewMemoryRepositories(t *testing.T) {
	repo := NewMemoryRepositories()
	assert.NotNil(t, repo.Users)
	assert.NotNil(t, repo.Secrets)
	assert.NotNil(t, repo.Sessions)
}

func TestMemoryUserRepository_Create(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	user := &models.User{
		ID:                 uuid.New(),
		Email:              "test@example.com",
		MasterPasswordHash: "hash123",
	}

	err := repo.Users.Create(ctx, user)
	require.NoError(t, err)
	assert.NotZero(t, user.CreatedAt)
	assert.NotZero(t, user.UpdatedAt)
}

func TestMemoryUserRepository_Create_Duplicate(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	user := &models.User{
		ID:    uuid.New(),
		Email: "dup@example.com",
	}

	err := repo.Users.Create(ctx, user)
	require.NoError(t, err)

	dup := &models.User{
		ID:    uuid.New(),
		Email: "dup@example.com",
	}

	err = repo.Users.Create(ctx, dup)
	assert.ErrorIs(t, err, domainerrors.ErrAlreadyExists)
}

func TestMemoryUserRepository_GetByID(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	user := &models.User{
		ID:    uuid.New(),
		Email: "get@example.com",
	}
	require.NoError(t, repo.Users.Create(ctx, user))

	got, err := repo.Users.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Email, got.Email)
}

func TestMemoryUserRepository_GetByID_NotFound(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	_, err := repo.Users.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemoryUserRepository_GetByEmail(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	user := &models.User{
		ID:    uuid.New(),
		Email: "byemail@example.com",
	}
	require.NoError(t, repo.Users.Create(ctx, user))

	got, err := repo.Users.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
}

func TestMemoryUserRepository_GetByEmail_NotFound(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	_, err := repo.Users.GetByEmail(ctx, "nonexistent@example.com")
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemoryUserRepository_GetByEmail_UserDeletedFromMap(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	user := &models.User{ID: uuid.New(), Email: "deleted@example.com"}
	require.NoError(t, repo.Users.Create(ctx, user))

	delete(repo.users, user.ID)

	_, err := repo.Users.GetByEmail(ctx, "deleted@example.com")
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemoryUserRepository_Update(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	user := &models.User{
		ID:    uuid.New(),
		Email: "update@example.com",
	}
	require.NoError(t, repo.Users.Create(ctx, user))

	user.Email = "updated@example.com"
	err := repo.Users.Update(ctx, user)
	require.NoError(t, err)

	got, err := repo.Users.GetByEmail(ctx, "updated@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)

	_, err = repo.Users.GetByEmail(ctx, "update@example.com")
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemoryUserRepository_Update_NotFound(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	err := repo.Users.Update(ctx, &models.User{ID: uuid.New()})
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemoryUserRepository_Update_SameEmail(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	user := &models.User{ID: uuid.New(), Email: "same@example.com"}
	require.NoError(t, repo.Users.Create(ctx, user))

	user.MasterPasswordHash = "newhash"
	err := repo.Users.Update(ctx, user)
	require.NoError(t, err)

	got, err := repo.Users.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "newhash", got.MasterPasswordHash)
}

func TestMemoryUserRepository_Delete(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	user := &models.User{
		ID:    uuid.New(),
		Email: "delete@example.com",
	}
	require.NoError(t, repo.Users.Create(ctx, user))

	err := repo.Users.Delete(ctx, user.ID)
	require.NoError(t, err)

	_, err = repo.Users.GetByID(ctx, user.ID)
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemoryUserRepository_Delete_NotFound(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	err := repo.Users.Delete(ctx, uuid.New())
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySecretRepository_Create(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	secret := &models.Secret{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   models.SecretCredential,
		Title:  "My Password",
	}

	err := repo.Secrets.Create(ctx, secret)
	require.NoError(t, err)
	assert.NotZero(t, secret.CreatedAt)
	assert.NotZero(t, secret.UpdatedAt)
}

func TestMemorySecretRepository_GetByID(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	secret := &models.Secret{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   models.SecretText,
		Title:  "Secret Note",
	}
	require.NoError(t, repo.Secrets.Create(ctx, secret))

	got, err := repo.Secrets.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, secret.Title, got.Title)
}

func TestMemorySecretRepository_GetByID_NotFound(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	_, err := repo.Secrets.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySecretRepository_GetByID_SoftDeleted(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	userID := uuid.New()
	secret := &models.Secret{
		ID:     uuid.New(),
		UserID: userID,
		Type:   models.SecretCard,
		Title:  "Deleted Card",
	}
	require.NoError(t, repo.Secrets.Create(ctx, secret))
	require.NoError(t, repo.Secrets.Delete(ctx, secret.ID, userID))

	_, err := repo.Secrets.GetByID(ctx, secret.ID)
	assert.ErrorIs(t, err, domainerrors.ErrNotFound, "soft-deleted secret should not be found by GetByID")
}

func TestMemorySecretRepository_ListByUser(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	userID := uuid.New()
	otherUserID := uuid.New()

	s1 := &models.Secret{ID: uuid.New(), UserID: userID, Type: models.SecretCredential, Title: "Login1"}
	s2 := &models.Secret{ID: uuid.New(), UserID: userID, Type: models.SecretCard, Title: "Card1"}
	s3 := &models.Secret{ID: uuid.New(), UserID: otherUserID, Type: models.SecretText, Title: "Other"}

	require.NoError(t, repo.Secrets.Create(ctx, s1))
	require.NoError(t, repo.Secrets.Create(ctx, s2))
	require.NoError(t, repo.Secrets.Create(ctx, s3))

	metas, err := repo.Secrets.ListByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, metas, 2)
}

func TestMemorySecretRepository_ListByUser_ExcludesDeleted(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	userID := uuid.New()

	s1 := &models.Secret{ID: uuid.New(), UserID: userID, Type: models.SecretCredential, Title: "Active"}
	s2 := &models.Secret{ID: uuid.New(), UserID: userID, Type: models.SecretText, Title: "Deleted"}

	require.NoError(t, repo.Secrets.Create(ctx, s1))
	require.NoError(t, repo.Secrets.Create(ctx, s2))
	require.NoError(t, repo.Secrets.Delete(ctx, s2.ID, userID))

	metas, err := repo.Secrets.ListByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, metas, 1)
	assert.Equal(t, "Active", metas[0].Title)
}

func TestMemorySecretRepository_ListByUser_Empty(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	metas, err := repo.Secrets.ListByUser(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, metas)
}

func TestMemorySecretRepository_Update(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	secret := &models.Secret{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   models.SecretBinary,
		Title:  "Original",
	}
	require.NoError(t, repo.Secrets.Create(ctx, secret))

	secret.Title = "Updated"
	err := repo.Secrets.Update(ctx, secret)
	require.NoError(t, err)

	got, err := repo.Secrets.GetByID(ctx, secret.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
}

func TestMemorySecretRepository_Update_NotFound(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	err := repo.Secrets.Update(ctx, &models.Secret{ID: uuid.New()})
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySecretRepository_Update_DeletedSecret(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	userID := uuid.New()
	secret := &models.Secret{ID: uuid.New(), UserID: userID, Type: models.SecretText, Title: "To Delete"}
	require.NoError(t, repo.Secrets.Create(ctx, secret))
	require.NoError(t, repo.Secrets.Delete(ctx, secret.ID, userID))

	secret.Title = "Updated After Delete"
	err := repo.Secrets.Update(ctx, secret)
	assert.ErrorIs(t, err, domainerrors.ErrNotFound, "updating deleted secret should fail")
}

func TestMemorySecretRepository_Update_PreservesCreatedAt(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	secret := &models.Secret{ID: uuid.New(), UserID: uuid.New(), Type: models.SecretText, Title: "Original"}
	require.NoError(t, repo.Secrets.Create(ctx, secret))

	createdAt := secret.CreatedAt
	time.Sleep(1 * time.Millisecond)

	secret.Title = "Updated"
	require.NoError(t, repo.Secrets.Update(ctx, secret))

	assert.Equal(t, createdAt, secret.CreatedAt, "CreatedAt should be preserved after update")
	assert.True(t, secret.UpdatedAt.After(createdAt), "UpdatedAt should be newer after update")
}

func TestMemorySecretRepository_Delete(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	userID := uuid.New()
	secret := &models.Secret{
		ID:     uuid.New(),
		UserID: userID,
		Type:   models.SecretCredential,
		Title:  "ToDelete",
	}
	require.NoError(t, repo.Secrets.Create(ctx, secret))

	err := repo.Secrets.Delete(ctx, secret.ID, userID)
	require.NoError(t, err)

	_, err = repo.Secrets.GetByID(ctx, secret.ID)
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySecretRepository_Delete_NotFound(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	err := repo.Secrets.Delete(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySecretRepository_Delete_WrongUser(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	secret := &models.Secret{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   models.SecretText,
		Title:  "Mine",
	}
	require.NoError(t, repo.Secrets.Create(ctx, secret))

	err := repo.Secrets.Delete(ctx, secret.ID, uuid.New())
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySecretRepository_Delete_Twice(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	userID := uuid.New()
	secret := &models.Secret{ID: uuid.New(), UserID: userID, Type: models.SecretText, Title: "Delete me twice"}
	require.NoError(t, repo.Secrets.Create(ctx, secret))
	require.NoError(t, repo.Secrets.Delete(ctx, secret.ID, userID))

	err := repo.Secrets.Delete(ctx, secret.ID, userID)
	assert.ErrorIs(t, err, domainerrors.ErrNotFound, "double delete should fail")
}

func TestMemorySessionRepository_CreateAndGet(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	session := &models.Session{
		Token:     "token-123",
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	err := repo.Sessions.Create(ctx, session)
	require.NoError(t, err)
	assert.NotZero(t, session.CreatedAt)

	got, err := repo.Sessions.GetByToken(ctx, "token-123")
	require.NoError(t, err)
	assert.Equal(t, session.UserID, got.UserID)
}

func TestMemorySessionRepository_GetByToken_NotFound(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	_, err := repo.Sessions.GetByToken(ctx, "nonexistent-token")
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySessionRepository_GetExpired(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	session := &models.Session{
		Token:     "expired-token",
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	require.NoError(t, repo.Sessions.Create(ctx, session))

	_, err := repo.Sessions.GetByToken(ctx, "expired-token")
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySessionRepository_Delete(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	session := &models.Session{
		Token:     "delete-me",
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Sessions.Create(ctx, session))

	err := repo.Sessions.Delete(ctx, "delete-me")
	require.NoError(t, err)

	_, err = repo.Sessions.GetByToken(ctx, "delete-me")
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySessionRepository_Delete_NotFound(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	err := repo.Sessions.Delete(ctx, "nonexistent")
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySessionRepository_CleanupExpired(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	active := &models.Session{
		Token:     "active",
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	expired := &models.Session{
		Token:     "expired",
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	require.NoError(t, repo.Sessions.Create(ctx, active))
	require.NoError(t, repo.Sessions.Create(ctx, expired))

	err := repo.Sessions.CleanupExpired(ctx, time.Now())
	require.NoError(t, err)

	_, err = repo.Sessions.GetByToken(ctx, "active")
	require.NoError(t, err)

	_, err = repo.Sessions.GetByToken(ctx, "expired")
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMemorySessionRepository_CleanupExpired_NoExpired(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	active := &models.Session{
		Token:     "active",
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Sessions.Create(ctx, active))

	err := repo.Sessions.CleanupExpired(ctx, time.Now())
	require.NoError(t, err)

	_, err = repo.Sessions.GetByToken(ctx, "active")
	require.NoError(t, err)
}

func TestMemorySessionRepository_CleanupExpired_Empty(t *testing.T) {
	repo := NewMemoryRepositories()
	ctx := context.Background()

	err := repo.Sessions.CleanupExpired(ctx, time.Now())
	require.NoError(t, err)
}
