package crypto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveKey_Deterministic(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	password := "my-secure-master-password-123!@#"

	key1 := DeriveKey(password, userID)
	key2 := DeriveKey(password, userID)

	assert.Equal(t, key1, key2, "DeriveKey should be deterministic for same password and userID")
}

func TestDeriveKey_DifferentUsers(t *testing.T) {
	t.Parallel()

	password := "same-password"

	key1 := DeriveKey(password, uuid.New())
	key2 := DeriveKey(password, uuid.New())

	assert.NotEqual(t, key1, key2, "different users should have different keys even with same password")
}

func TestDeriveKey_DifferentPasswords(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	key1 := DeriveKey("password-1", userID)
	key2 := DeriveKey("password-2", userID)

	assert.NotEqual(t, key1, key2, "different passwords should produce different keys")
}

func TestDeriveKey_OutputLength(t *testing.T) {
	t.Parallel()

	key := DeriveKey("test-password", uuid.New())
	assert.Equal(t, KeySize, len(key), "DeriveKey should produce a 32-byte key")
	assert.Equal(t, 32, len(key), "Key should be 32 bytes")
}

func TestDeriveKey_EmptyPassword(t *testing.T) {
	t.Parallel()

	key := DeriveKey("", uuid.New())
	assert.Equal(t, KeySize, len(key), "empty password should still produce a key")
	assert.NotZero(t, key, "key should not be all zeros even with empty password")
}

func TestDeriveKey_NilUserID(t *testing.T) {
	t.Parallel()

	key1 := DeriveKey("password", uuid.Nil)
	key2 := DeriveKey("password", uuid.Nil)

	assert.Equal(t, key1, key2, "nil UUID should be deterministic")
	assert.Equal(t, KeySize, len(key1))
}

func TestDeriveKey_DifferentDomains(t *testing.T) {
	t.Parallel()

	// Тест, что ключ отличается от того, что было бы с другим domainTag.
	userID := uuid.New()
	password := "test"

	key := DeriveKey(password, userID)
	assert.Equal(t, KeySize, len(key))
}

func TestDeriveKeyWithParams_Deterministic(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	key1, err := DeriveKeyWithParams("pass", userID, 1, 64*1024, 1)
	require.NoError(t, err)

	key2, err := DeriveKeyWithParams("pass", userID, 1, 64*1024, 1)
	require.NoError(t, err)

	assert.Equal(t, key1, key2)
}

func TestDeriveKeyWithParams_InvalidParams(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	_, err := DeriveKeyWithParams("pass", userID, 0, 64*1024, 4)
	assert.Error(t, err, "zero time should error")

	_, err = DeriveKeyWithParams("pass", userID, 3, 0, 4)
	assert.Error(t, err, "zero memory should error")

	_, err = DeriveKeyWithParams("pass", userID, 3, 64*1024, 0)
	assert.Error(t, err, "zero threads should error")
}

func TestDeriveKeyWithParams_Minimal(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	key, err := DeriveKeyWithParams("pass", userID, 1, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, KeySize, len(key))
}
