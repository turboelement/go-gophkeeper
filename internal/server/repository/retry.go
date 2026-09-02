package repository

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// DoWithRetries выполняет переданную операцию с экспоненциальной задержкой и jitter.
// Повторяет только те ошибки, для которых isRetryable возвращает true.
func DoWithRetries(ctx context.Context, op func() error, cfg RetryConfig) error {
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := float64(cfg.BaseDelay) * math.Pow(cfg.Multiplier, float64(attempt-1))
			delay = math.Min(delay, float64(cfg.MaxDelay))
			jitter := (rand.Float64()*2 - 1) * cfg.JitterFactor
			delay = delay * (1 + jitter)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(delay)):
			}
		}

		err := op()
		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetryable(err) {
			return mapError(err)
		}
	}

	return fmt.Errorf("all %d attempts failed, last error: %w", cfg.MaxAttempts, lastErr)
}
