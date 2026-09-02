package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESGCMEngine_EncryptDecrypt_RoundTrip(t *testing.T) {
	t.Parallel()

	key := testKey()
	engine := NewAESGCMEngine(key)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "empty data",
			plaintext: []byte{},
		},
		{
			name:      "small data",
			plaintext: []byte("hello, world!"),
		},
		{
			name:      "large data (64KB)",
			plaintext: bytes.Repeat([]byte("A"), 64*1024),
		},
		{
			name:      "binary data with null bytes",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x00, 0x7F},
		},
		{
			name:      "unicode text",
			plaintext: []byte("hello, GophKeeper! 🚀"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encrypted, err := engine.Encrypt(tt.plaintext)
			require.NoError(t, err, "Encrypt should not error")

			expectedMinLen := NonceSize + 16
			assert.GreaterOrEqual(t, len(encrypted), len(tt.plaintext)+expectedMinLen,
				"ciphertext should be longer than plaintext")

			if len(tt.plaintext) > 0 {
				assert.NotEqual(t, tt.plaintext, encrypted[:len(tt.plaintext)],
					"ciphertext should not contain plaintext prefix")
			}

			decrypted, err := engine.Decrypt(encrypted)
			require.NoError(t, err, "Decrypt should not error")

			assert.True(t, bytes.Equal(tt.plaintext, decrypted),
				"decrypted data should match original plaintext")
		})
	}
}

func TestAESGCMEngine_UniqueNonce(t *testing.T) {
	t.Parallel()

	key := testKey()
	engine := NewAESGCMEngine(key)
	plaintext := []byte("same data")

	results := make([][]byte, 10)
	for i := range results {
		enc, err := engine.Encrypt(plaintext)
		require.NoError(t, err)
		results[i] = enc
	}

	for i := 1; i < len(results); i++ {
		assert.False(t, bytes.Equal(results[0], results[i]),
			"encrypting same data twice should produce different ciphertext (unique nonce)")
	}
}

func TestAESGCMEngine_Decrypt_TamperedData(t *testing.T) {
	t.Parallel()

	key := testKey()
	engine := NewAESGCMEngine(key)
	plaintext := []byte("sensitive data")

	encrypted, err := engine.Encrypt(plaintext)
	require.NoError(t, err)

	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[len(tampered)/2] ^= 0xFF

	_, err = engine.Decrypt(tampered)
	assert.Error(t, err, "should detect tampered ciphertext")
	assert.Contains(t, err.Error(), "decrypt", "error should mention decryption failure")
}

func TestAESGCMEngine_Decrypt_ShortData(t *testing.T) {
	t.Parallel()

	key := testKey()
	engine := NewAESGCMEngine(key)

	_, err := engine.Decrypt([]byte{})
	assert.Error(t, err, "empty ciphertext should error")

	_, err = engine.Decrypt([]byte{0x01, 0x02, 0x03})
	assert.Error(t, err, "ciphertext shorter than nonce should error")
}

func TestAESGCMEngine_DifferentKeys(t *testing.T) {
	t.Parallel()

	key1 := testKey()
	key2 := testKey()
	plaintext := []byte("secret data")

	engine1 := NewAESGCMEngine(key1)
	engine2 := NewAESGCMEngine(key2)

	encrypted1, err := engine1.Encrypt(plaintext)
	require.NoError(t, err)

	_, err = engine2.Decrypt(encrypted1)
	assert.Error(t, err, "decrypting with different key should fail")
}

func TestAESGCMEngine_ConcurrentUse(t *testing.T) {
	t.Parallel()

	key := testKey()
	engine := NewAESGCMEngine(key)

	const goroutines = 20
	errCh := make(chan error, goroutines)

	for range goroutines {
		go func() {
			plaintext := make([]byte, 128)
			_, _ = rand.Read(plaintext)

			enc, err := engine.Encrypt(plaintext)
			if err != nil {
				errCh <- err
				return
			}

			dec, err := engine.Decrypt(enc)
			if err != nil {
				errCh <- err
				return
			}

			if !bytes.Equal(plaintext, dec) {
				errCh <- assert.AnError
				return
			}

			errCh <- nil
		}()
	}

	for range goroutines {
		err := <-errCh
		assert.NoError(t, err, "concurrent encrypt/decrypt should not race")
	}
}

func testKey() [KeySize]byte {
	var key [KeySize]byte
	_, _ = rand.Read(key[:])
	return key
}

func TestIntegration_EncryptDecryptWithDerivedKey(t *testing.T) {
	t.Parallel()

	password := "correct-horse-battery-staple"
	userID := uuid.New()
	secretData := []byte(`{"login":"user@example.com","password":"supersecret"}`)

	key := DeriveKey(password, userID)
	engine := NewAESGCMEngine(key)

	encrypted, err := engine.Encrypt(secretData)
	require.NoError(t, err)

	decrypted, err := engine.Decrypt(encrypted)
	require.NoError(t, err)

	assert.Equal(t, secretData, decrypted, "full integration round-trip should work")
}
