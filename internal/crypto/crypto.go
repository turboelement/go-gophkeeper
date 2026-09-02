// Package crypto предоставляет интерфейс и реализацию клиент-сайд шифрования
// для GophKeeper. Все операции шифрования выполняются исключительно на стороне
// клиента — сервер никогда не получает незашифрованные данные.
//
// Архитектурные решения:
//   - Zero Trust: сервер оперирует только EncryptedPayload []byte как непрозрачным blob
//   - Алгоритм: AES-256-GCM (предпочтительно) или NaCl secretbox (альтернатива)
//   - Ключ: 256-битный MasterKey, производный от мастер-пароля через Argon2id
//   - Nonce: 12 байт, crypto/rand, уникален для каждого шифрования
//   - Формат payload: [12B nonce][ciphertext + 16B GCM tag]
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	// NonceSize — размер nonce для AES-GCM (12 байт, рекомендуется NIST).
	NonceSize = 12

	// KeySize — размер ключа AES-256 (32 байта).
	KeySize = 32
)

// CryptoEngine — контракт шифрования/дешифрования данных.
// Реализация должна быть горутино-безопасной (не хранить состояние между вызовами).
type CryptoEngine interface {
	// Encrypt шифрует plaintext и возвращает ciphertext, готовый для хранения.
	// Формат ciphertext: [12B nonce][GCM-encrypted data].
	// Nonce генерируется автоматически через crypto/rand.
	Encrypt(plaintext []byte) (ciphertext []byte, err error)

	// Decrypt дешифрует ciphertext (в формате, возвращённом Encrypt).
	// Извлекает nonce из первых 12 байт и проверяет GCM-тег аутентификации.
	Decrypt(ciphertext []byte) (plaintext []byte, err error)
}

// AESGCMEngine — реализация CryptoEngine на основе AES-256-GCM.
// ГорУтино-безопасность: все методы используют только локальные переменные
// и переданный ключ (не мутируют receiver).
type AESGCMEngine struct {
	key [KeySize]byte
}

// NewAESGCMEngine создаёт AESGCMEngine с указанным 256-битным ключом.
// key должен быть длиной ровно KeySize (32 байта).
// Паникует, если длина key != KeySize (programming error).
func NewAESGCMEngine(key [KeySize]byte) *AESGCMEngine {
	return &AESGCMEngine{key: key}
}

// Encrypt шифрует данные с помощью AES-256-GCM.
//  1. Генерирует 12-байтовый nonce через crypto/rand.
//  2. Шифрует с GCM (без дополнительных данных, associatedData = nil).
//  3. Возвращает nonce || ciphertext (где ciphertext уже включает GCM-tag).
func (e *AESGCMEngine) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: generate nonce: %w", err)
	}

	// Seal(dst, nonce, plaintext, additionalData)
	// nonce прикрепляется к выходу: сначала nonce, затем зашифрованный текст.
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt дешифрует данные, зашифрованные Encrypt.
//  1. Извлекает первые 12 байт как nonce.
//  2. Дешифрует с проверкой GCM-тега аутентификации.
//  3. Возвращает plaintext или ошибку, если данные были изменены.
func (e *AESGCMEngine) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < NonceSize+1 {
		return nil, fmt.Errorf("crypto: ciphertext too short: %d bytes", len(ciphertext))
	}

	block, err := aes.NewCipher(e.key[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: create gcm: %w", err)
	}

	nonce := ciphertext[:NonceSize]
	encryptedData := ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: %w", err)
	}

	return plaintext, nil
}

// Ensure interfaces compliance.
var _ CryptoEngine = (*AESGCMEngine)(nil)
