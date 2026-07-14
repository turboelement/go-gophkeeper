package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SecretType — тип хранимого секрета.
type SecretType string

const (
	SecretCredential SecretType = "credential"
	SecretText       SecretType = "text"
	SecretCard       SecretType = "card"
	SecretBinary     SecretType = "binary"
)

// Secret — зашифрованный секрет, хранящийся на сервере.
// Сервер никогда не имеет доступа к незашифрованным данным.
type Secret struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	Type             SecretType
	Title            string
	Metadata         json.RawMessage
	EncryptedPayload []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

// SecretMeta — облегчённое представление секрета без зашифрованной нагрузки.
// Используется при получении списка секретов пользователя.
type SecretMeta struct {
	ID        uuid.UUID
	Type      SecretType
	Title     string
	CreatedAt time.Time
}
