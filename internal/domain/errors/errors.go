// Package errors определяет базовые sentinel-ошибки, используемые во всех слоях приложения.
package errors

import "errors"

var (
	// ErrNotFound возвращается, когда запись не найдена.
	ErrNotFound = errors.New("record not found")

	// ErrAlreadyExists возвращается при попытке создать дубликат записи.
	ErrAlreadyExists = errors.New("record already exists")

	// ErrInvalidCredentials возвращается при неверном email или пароле.
	ErrInvalidCredentials = errors.New("invalid email or password")
)
