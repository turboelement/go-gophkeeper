package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()
	assert.Equal(t, int32(25), cfg.MaxConns)
	assert.Equal(t, int32(2), cfg.MinConns)
	assert.Equal(t, 30*time.Minute, cfg.MaxConnLifetime)
	assert.Equal(t, 5*time.Minute, cfg.MaxConnIdleTime)
	assert.Equal(t, 1*time.Minute, cfg.HealthCheckPeriod)
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	assert.Equal(t, 3, cfg.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, cfg.BaseDelay)
	assert.Equal(t, 2*time.Second, cfg.MaxDelay)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.InDelta(t, 0.2, cfg.JitterFactor, 0.001)
}
