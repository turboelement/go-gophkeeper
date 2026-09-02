package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidate(t *testing.T) {
	userID := uuid.New()
	secret := "my-secret-key-12345"
	expiration := 30 * time.Minute

	token, err := Generate(userID, secret, expiration)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := Validate(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
}

func TestValidate_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	secret := "my-secret-key-12345"

	token, err := Generate(userID, secret, -1*time.Second)
	require.NoError(t, err)

	_, err = Validate(token, secret)
	assert.Error(t, err)
}

func TestValidate_WrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := Generate(userID, "correct-secret", 30*time.Minute)
	require.NoError(t, err)

	_, err = Validate(token, "wrong-secret")
	assert.Error(t, err)
}

func TestValidate_InvalidToken(t *testing.T) {
	_, err := Validate("invalid-token-string", "secret")
	assert.Error(t, err)
}

func TestGenerate_EmptyUserID(t *testing.T) {
	secret := "secret"
	token, err := Generate(uuid.Nil, secret, 30*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := Validate(token, secret)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, claims.UserID)
}

func TestGenerate_EmptySecret(t *testing.T) {
	userID := uuid.New()
	token, err := Generate(userID, "", 30*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidate_InvalidSignature(t *testing.T) {
	userID := uuid.New()
	secret := "valid-secret"

	token, err := Generate(userID, secret, 30*time.Minute)
	require.NoError(t, err)

	parts := []byte(token)
	parts[len(parts)-5] ^= 0xFF
	tamperedToken := string(parts)

	_, err = Validate(tamperedToken, secret)
	assert.Error(t, err)
}

func TestClaims_RegisteredClaims(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	expiration := 1 * time.Hour

	token, err := Generate(userID, secret, expiration)
	require.NoError(t, err)

	claims, err := Validate(token, secret)
	require.NoError(t, err)

	assert.Equal(t, userID.String(), claims.Subject)
	assert.WithinRange(t, claims.IssuedAt.Time, time.Now().Add(-time.Minute), time.Now().Add(time.Minute))
	assert.WithinRange(t, claims.ExpiresAt.Time, time.Now().Add(expiration-time.Minute), time.Now().Add(expiration+time.Minute))
}
