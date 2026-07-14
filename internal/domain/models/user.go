// Package models содержит базовые доменные сущности GophKeeper.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User — зарегистрированный пользователь системы.
type User struct {
	ID                 uuid.UUID
	Email              string
	MasterPasswordHash string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
