package repository

import (
	"errors"
	"io"
	"syscall"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	domainerrors "go-gophkeeper/internal/domain/errors"
)

// retryableCodes содержит коды ошибок PostgreSQL, которые можно безопасно повторять.
var retryableCodes = map[string]bool{
	pgerrcode.SerializationFailure: true, // 40001
	pgerrcode.DeadlockDetected:     true, // 40P01
}

// mapError преобразует ошибку PostgreSQL или pgx в sentinel-ошибку доменного уровня.
// Должна вызываться только для ошибок, не подлежащих повторным попыткам.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return domainerrors.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return domainerrors.ErrAlreadyExists
		case pgerrcode.ForeignKeyViolation:
			return domainerrors.ErrNotFound
		}
	}

	return err
}

// isRetryable определяет, является ли ошибка временной и может быть повторена.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if retryableCodes[pgErr.Code] {
			return true
		}
	}

	return errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET)
}
