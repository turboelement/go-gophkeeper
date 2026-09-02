// Package repository содержит адаптеры доступа к данным (PostgreSQL и in-memory).
package repository

import "time"

// DatabasePoolConfig — настройки пула соединений PostgreSQL.
type DatabasePoolConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// DefaultPoolConfig возвращает DatabasePoolConfig с разумными значениями по умолчанию.
func DefaultPoolConfig() DatabasePoolConfig {
	return DatabasePoolConfig{
		MaxConns:          25,
		MinConns:          2,
		MaxConnLifetime:   30 * time.Minute,
		MaxConnIdleTime:   5 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}
}

// RetryConfig — настройки политики повторных попыток с экспоненциальной задержкой.
type RetryConfig struct {
	MaxAttempts  int
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	JitterFactor float64
}

// DefaultRetryConfig возвращает RetryConfig с разумными значениями по умолчанию.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.2,
	}
}
