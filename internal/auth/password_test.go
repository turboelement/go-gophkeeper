package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	password := "test-password-123"

	hash, err := Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = Verify(hash, password)
	assert.NoError(t, err)
}

func TestHash_DifferentPasswords_DifferentHashes(t *testing.T) {
	hash1, err := Hash("password-1")
	require.NoError(t, err)

	hash2, err := Hash("password-2")
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)
}

func TestHash_SamePassword_DifferentHashes(t *testing.T) {
	// bcrypt с разными солями даёт разные хеши для одного пароля.
	hash1, err := Hash("same-password")
	require.NoError(t, err)

	hash2, err := Hash("same-password")
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)
}

func TestVerify_CorrectPassword(t *testing.T) {
	password := "test-password-123"
	hash, err := Hash(password)
	require.NoError(t, err)

	err = Verify(hash, password)
	assert.NoError(t, err)
}

func TestVerify_WrongPassword(t *testing.T) {
	hash, err := Hash("correct-password")
	require.NoError(t, err)

	err = Verify(hash, "wrong-password")
	assert.Error(t, err)
}

func TestVerify_EmptyPassword(t *testing.T) {
	hash, err := Hash("some-password")
	require.NoError(t, err)

	err = Verify(hash, "")
	assert.Error(t, err)
}

func TestVerify_EmptyHash(t *testing.T) {
	err := Verify("", "password")
	assert.Error(t, err)
}

func TestVerify_InvalidHash(t *testing.T) {
	err := Verify("not-a-valid-bcrypt-hash", "password")
	assert.Error(t, err)
}

func TestVerify_EmptyBoth(t *testing.T) {
	err := Verify("", "")
	assert.Error(t, err)
}

func TestHash_EmptyPassword(t *testing.T) {
	// Пустой пароль технически можно захешировать (хотя не рекомендуется).
	hash, err := Hash("")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = Verify(hash, "")
	assert.NoError(t, err)
}

func TestHash_LongPassword(t *testing.T) {
	password := string(make([]byte, 72)) // bcrypt max is 72 bytes
	hash, err := Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = Verify(hash, password)
	assert.NoError(t, err)
}
