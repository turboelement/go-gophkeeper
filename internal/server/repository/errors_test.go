package repository

import (
	"errors"
	"io"
	"syscall"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	domainerrors "go-gophkeeper/internal/domain/errors"
)

func TestMapError_Nil(t *testing.T) {
	assert.Nil(t, mapError(nil))
}

func TestMapError_ErrNoRows(t *testing.T) {
	err := mapError(pgx.ErrNoRows)
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMapError_UniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation}
	err := mapError(pgErr)
	assert.ErrorIs(t, err, domainerrors.ErrAlreadyExists)
}

func TestMapError_ForeignKeyViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}
	err := mapError(pgErr)
	assert.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMapError_UnknownPgError(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "XX000"} // internal error
	err := mapError(pgErr)
	assert.Equal(t, pgErr, err)
}

func TestMapError_GenericError(t *testing.T) {
	genericErr := errors.New("some generic error")
	err := mapError(genericErr)
	assert.Equal(t, genericErr, err)
}

func TestMapError_WrappedErrNoRows(t *testing.T) {
	wrapped := errors.New("wrapped: something")
	err := mapError(wrapped)
	assert.Equal(t, wrapped, err)
}

func TestIsRetryable_Nil(t *testing.T) {
	assert.False(t, isRetryable(nil))
}

func TestIsRetryable_SerializationFailure(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.SerializationFailure}
	assert.True(t, isRetryable(pgErr))
}

func TestIsRetryable_DeadlockDetected(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.DeadlockDetected}
	assert.True(t, isRetryable(pgErr))
}

func TestIsRetryable_UniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: pgerrcode.UniqueViolation}
	assert.False(t, isRetryable(pgErr))
}

func TestIsRetryable_IOError(t *testing.T) {
	assert.True(t, isRetryable(io.EOF))
}

func TestIsRetryable_ConnectionReset(t *testing.T) {
	assert.True(t, isRetryable(syscall.ECONNRESET))
}

func TestIsRetryable_GenericError(t *testing.T) {
	assert.False(t, isRetryable(errors.New("generic error")))
}
