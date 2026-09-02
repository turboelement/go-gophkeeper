// Package auth предоставляет функции для хеширования паролей (bcrypt)
// и работы с JWT-токенами.
package auth

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// Hash возвращает bcrypt-хеш пароля.
func Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify сравнивает пароль с bcrypt-хешем.
// Возвращает nil при совпадении, ошибку — если не совпал.
func Verify(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
