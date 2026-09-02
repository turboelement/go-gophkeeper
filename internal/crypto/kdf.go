package crypto

import (
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

const (
	// domainTag — доменная константа для salt (не даёт использовать KDF вне контекста GophKeeper).
	domainTag = "go-gophkeeper-v1"

	// argon2Time — количество итераций Argon2id (time cost).
	argon2Time = 3

	// argon2Memory — объём памяти в KiB (64 MB).
	argon2Memory = 64 * 1024

	// argon2Threads — степень параллелизма.
	argon2Threads = 4

	// argon2KeyLen — длина выходного ключа AES-256 (32 байта).
	argon2KeyLen = 32
)

// DeriveKey вычисляет 256-битный ключ шифрования из мастер-пароля и UserID.
//
// Алгоритм: Argon2id(password, salt, time=3, memory=64MB, threads=4)
//
// Salt формируется как SHA-256(userID_Bytes + domainTag), что даёт:
//   - Уникальность между пользователями (разные UserID → разный salt)
//   - Детерминированность (один пользователь всегда получает тот же ключ)
//   - Доменная изоляция (domainTag не даёт использовать KDF вне GophKeeper)
//
// Параметры Argon2id (time=3, memory=64MB) обеспечивают защиту от GPU-брутфорса
// при сохранении приемлемого времени выполнения (~1-3 секунды на современном CPU).
//
// ВАЖНО: ключ НЕ хранится на диске. Он вычисляется при каждом запуске клиента
// и существует только в памяти процесса.
func DeriveKey(password string, userID uuid.UUID) [KeySize]byte {
	// Формируем salt: SHA-256(userID_16bytes + domainTag).
	// Использование SHA-256 гарантирует, что salt имеет ровно 32 байта
	// (рекомендуемый размер salt для Argon2id).
	saltInput := make([]byte, 0, 16+len(domainTag))
	saltInput = append(saltInput, userID[:]...)
	saltInput = append(saltInput, domainTag...)

	salt := sha256.Sum256(saltInput)

	// argon2.IDKey принимает time uint32, memory uint32, threads uint8.
	keyBytes := argon2.IDKey([]byte(password), salt[:], argon2Time, argon2Memory, uint8(argon2Threads), argon2KeyLen)

	var key [KeySize]byte
	copy(key[:], keyBytes)
	return key
}

// DeriveKeyWithParams — как DeriveKey, но с кастомными параметрами Argon2id.
// Используется для тестов или тонкой настройки производительности/безопасности.
func DeriveKeyWithParams(password string, userID uuid.UUID, time, memory, threads uint32) ([KeySize]byte, error) {
	if time == 0 || memory == 0 || threads == 0 {
		return [KeySize]byte{}, fmt.Errorf("crypto: argon2 params must be positive, got time=%d, memory=%d, threads=%d",
			time, memory, threads)
	}

	saltInput := make([]byte, 0, 16+len(domainTag))
	saltInput = append(saltInput, userID[:]...)
	saltInput = append(saltInput, domainTag...)

	salt := sha256.Sum256(saltInput)

	keyBytes := argon2.IDKey([]byte(password), salt[:], time, memory, uint8(threads), argon2KeyLen)

	var key [KeySize]byte
	copy(key[:], keyBytes)
	return key, nil
}
