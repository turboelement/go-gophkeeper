package repository

import (
	"context"
	"errors"
	"io"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoWithRetries_SuccessFirstAttempt(t *testing.T) {
	attempts := 0
	err := DoWithRetries(context.Background(), func() error {
		attempts++
		return nil
	}, RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestDoWithRetries_SuccessAfterRetries(t *testing.T) {
	attempts := 0
	err := DoWithRetries(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return io.EOF // retryable
		}
		return nil
	}, RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestDoWithRetries_AllRetriesFailed(t *testing.T) {
	attempts := 0
	err := DoWithRetries(context.Background(), func() error {
		attempts++
		return io.EOF // retryable
	}, RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all 3 attempts failed")
	assert.Equal(t, 3, attempts)
}

func TestDoWithRetries_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	err := DoWithRetries(ctx, func() error {
		cancel() // отменяем контекст при первом вызове
		return io.EOF // retryable
	}, RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	})

	assert.ErrorIs(t, err, context.Canceled)
}

func TestDoWithRetries_NonRetryableError(t *testing.T) {
	sentinelErr := errors.New("non-retryable")
	attempts := 0
	err := DoWithRetries(context.Background(), func() error {
		attempts++
		return sentinelErr
	}, RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	})

	assert.Error(t, err)
	assert.Equal(t, 1, attempts, "non-retryable should stop after 1 attempt")
}

func TestDoWithRetries_RetryableIOError(t *testing.T) {
	attempts := 0
	err := DoWithRetries(context.Background(), func() error {
		attempts++
		if attempts == 1 {
			return io.EOF // retryable
		}
		return nil
	}, RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestDoWithRetries_RetryableConnectionReset(t *testing.T) {
	attempts := 0
	err := DoWithRetries(context.Background(), func() error {
		attempts++
		if attempts == 1 {
			return syscall.ECONNRESET // retryable
		}
		return nil
	}, RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestDoWithRetries_DefaultConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	assert.Equal(t, 3, cfg.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, cfg.BaseDelay)
	assert.Equal(t, 2*time.Second, cfg.MaxDelay)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.InDelta(t, 0.2, cfg.JitterFactor, 0.01)
}

func TestDoWithRetries_ZeroAttempts(t *testing.T) {
	err := DoWithRetries(context.Background(), func() error {
		return io.EOF
	}, RetryConfig{
		MaxAttempts: 0,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	})

	require.Error(t, err)
}
