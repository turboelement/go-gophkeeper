package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-gophkeeper/internal/domain/interfaces"
	"go-gophkeeper/internal/domain/models"
)

// PostgresRepositories группирует все реализации репозиториев на PostgreSQL.
// Владеет единым пулом соединений, общим для всех репозиториев.
type PostgresRepositories struct {
	pool     *pgxpool.Pool
	retryCfg RetryConfig

	Users    interfaces.UserRepository
	Secrets  interfaces.SecretRepository
	Sessions interfaces.SessionRepository
}

// NewPostgresRepositories создаёт PostgresRepositories, подключаясь к базе,
// применяя миграции и возвращая готовый экземпляр.
func NewPostgresRepositories(
	ctx context.Context,
	dsn string,
	poolCfg DatabasePoolConfig,
	retryCfg RetryConfig,
) (*PostgresRepositories, error) {
	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("database migrations: %w", err)
	}

	pgxCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing DSN: %w", err)
	}

	pgxCfg.MaxConns = poolCfg.MaxConns
	pgxCfg.MinConns = poolCfg.MinConns
	pgxCfg.MaxConnLifetime = poolCfg.MaxConnLifetime
	pgxCfg.MaxConnIdleTime = poolCfg.MaxConnIdleTime
	pgxCfg.HealthCheckPeriod = poolCfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	ur := &postgresUserRepository{pool: pool, retryCfg: retryCfg}
	sr := &postgresSecretRepository{pool: pool, retryCfg: retryCfg}
	ssr := &postgresSessionRepository{pool: pool, retryCfg: retryCfg}

	return &PostgresRepositories{
		pool:     pool,
		retryCfg: retryCfg,
		Users:    ur,
		Secrets:  sr,
		Sessions: ssr,
	}, nil
}

// Close закрывает пул соединений.
func (r *PostgresRepositories) Close() {
	r.pool.Close()
}

func runMigrations(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("creating migration instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("applying migrations: %w", err)
	}

	return nil
}

// ----- postgresUserRepository -----

type postgresUserRepository struct {
	pool     *pgxpool.Pool
	retryCfg RetryConfig
}

func (r *postgresUserRepository) Create(ctx context.Context, user *models.User) error {
	return DoWithRetries(ctx, func() error {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO users (id, email, master_password_hash, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			user.ID, user.Email, user.MasterPasswordHash, user.CreatedAt, user.UpdatedAt,
		)
		return err
	}, r.retryCfg)
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := DoWithRetries(ctx, func() error {
		return r.pool.QueryRow(ctx,
			`SELECT id, email, master_password_hash, created_at, updated_at FROM users WHERE id = $1`,
			id,
		).Scan(&user.ID, &user.Email, &user.MasterPasswordHash, &user.CreatedAt, &user.UpdatedAt)
	}, r.retryCfg)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := DoWithRetries(ctx, func() error {
		return r.pool.QueryRow(ctx,
			`SELECT id, email, master_password_hash, created_at, updated_at FROM users WHERE email = $1`,
			email,
		).Scan(&user.ID, &user.Email, &user.MasterPasswordHash, &user.CreatedAt, &user.UpdatedAt)
	}, r.retryCfg)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresUserRepository) Update(ctx context.Context, user *models.User) error {
	return DoWithRetries(ctx, func() error {
		_, err := r.pool.Exec(ctx,
			`UPDATE users SET email = $1, master_password_hash = $2, updated_at = $3 WHERE id = $4`,
			user.Email, user.MasterPasswordHash, user.UpdatedAt, user.ID,
		)
		return err
	}, r.retryCfg)
}

func (r *postgresUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return DoWithRetries(ctx, func() error {
		_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
		return err
	}, r.retryCfg)
}

// ----- postgresSecretRepository -----

type postgresSecretRepository struct {
	pool     *pgxpool.Pool
	retryCfg RetryConfig
}

func (r *postgresSecretRepository) Create(ctx context.Context, secret *models.Secret) error {
	return DoWithRetries(ctx, func() error {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO secrets (id, user_id, type, title, metadata, encrypted_payload, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			secret.ID, secret.UserID, secret.Type, secret.Title,
			secret.Metadata, secret.EncryptedPayload,
			secret.CreatedAt, secret.UpdatedAt,
		)
		return err
	}, r.retryCfg)
}

func (r *postgresSecretRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Secret, error) {
	var secret models.Secret
	err := DoWithRetries(ctx, func() error {
		return r.pool.QueryRow(ctx,
			`SELECT id, user_id, type, title, metadata, encrypted_payload, created_at, updated_at, deleted_at
			 FROM secrets WHERE id = $1 AND deleted_at IS NULL`,
			id,
		).Scan(&secret.ID, &secret.UserID, &secret.Type, &secret.Title,
			&secret.Metadata, &secret.EncryptedPayload,
			&secret.CreatedAt, &secret.UpdatedAt, &secret.DeletedAt)
	}, r.retryCfg)

	if err != nil {
		return nil, err
	}
	return &secret, nil
}

func (r *postgresSecretRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.SecretMeta, error) {
	var metas []models.SecretMeta

	err := DoWithRetries(ctx, func() error {
		rows, err := r.pool.Query(ctx,
			`SELECT id, type, title, created_at FROM secrets
			 WHERE user_id = $1 AND deleted_at IS NULL
			 ORDER BY created_at DESC`,
			userID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		metas = make([]models.SecretMeta, 0)
		for rows.Next() {
			var m models.SecretMeta
			if err := rows.Scan(&m.ID, &m.Type, &m.Title, &m.CreatedAt); err != nil {
				return err
			}
			metas = append(metas, m)
		}
		return rows.Err()
	}, r.retryCfg)

	if err != nil {
		return nil, err
	}
	return metas, nil
}

func (r *postgresSecretRepository) Update(ctx context.Context, secret *models.Secret) error {
	return DoWithRetries(ctx, func() error {
		_, err := r.pool.Exec(ctx,
			`UPDATE secrets SET type = $1, title = $2, metadata = $3, encrypted_payload = $4, updated_at = $5
			 WHERE id = $6 AND user_id = $7 AND deleted_at IS NULL`,
			secret.Type, secret.Title, secret.Metadata, secret.EncryptedPayload,
			secret.UpdatedAt, secret.ID, secret.UserID,
		)
		return err
	}, r.retryCfg)
}

func (r *postgresSecretRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return DoWithRetries(ctx, func() error {
		_, err := r.pool.Exec(ctx,
			`UPDATE secrets SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND user_id = $3 AND deleted_at IS NULL`,
			time.Now(), id, userID,
		)
		return err
	}, r.retryCfg)
}

// ----- postgresSessionRepository -----

type postgresSessionRepository struct {
	pool     *pgxpool.Pool
	retryCfg RetryConfig
}

func (r *postgresSessionRepository) Create(ctx context.Context, session *models.Session) error {
	return DoWithRetries(ctx, func() error {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO sessions (token, user_id, expires_at, created_at)
			 VALUES ($1, $2, $3, $4)`,
			session.Token, session.UserID, session.ExpiresAt, session.CreatedAt,
		)
		return err
	}, r.retryCfg)
}

func (r *postgresSessionRepository) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	var session models.Session
	err := DoWithRetries(ctx, func() error {
		return r.pool.QueryRow(ctx,
			`SELECT token, user_id, expires_at, created_at FROM sessions
			 WHERE token = $1 AND expires_at > NOW()`,
			token,
		).Scan(&session.Token, &session.UserID, &session.ExpiresAt, &session.CreatedAt)
	}, r.retryCfg)

	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *postgresSessionRepository) Delete(ctx context.Context, token string) error {
	return DoWithRetries(ctx, func() error {
		_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
		return err
	}, r.retryCfg)
}

func (r *postgresSessionRepository) CleanupExpired(ctx context.Context, before time.Time) error {
	return DoWithRetries(ctx, func() error {
		_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < $1`, before)
		return err
	}, r.retryCfg)
}
