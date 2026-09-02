// Package app выполняет сборку зависимостей, настройку роутера
// и управление жизненным циклом сервера (graceful shutdown).
package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"go-gophkeeper/internal/domain/interfaces"
	"go-gophkeeper/internal/pkg/response"
	"go-gophkeeper/internal/server/config"
	"go-gophkeeper/internal/server/handler"
	"go-gophkeeper/internal/server/middleware"
	"go-gophkeeper/internal/server/repository"
	"go-gophkeeper/internal/server/service"
)

// repoCloser — минимальный интерфейс для закрытия репозиториев.
type repoCloser interface {
	Close()
}

// App — собранное приложение сервера.
type App struct {
	Server *http.Server
	Logger *zap.Logger
	closer repoCloser
}

// New создаёт и собирает приложение из конфига и логера.
func New(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*App, error) {
	// 1. Выбор хранилища: PostgreSQL или in-memory.
	var userRepo interfaces.UserRepository
	var sessionRepo interfaces.SessionRepository
	var secretRepo interfaces.SecretRepository
	var closer repoCloser

	if cfg.Database.DSN != "" {
		poolCfg := repository.DatabasePoolConfig{
			MaxConns:          int32(cfg.Database.MaxOpenConns),
			MinConns:          int32(cfg.Database.MaxIdleConns),
			MaxConnLifetime:   cfg.Database.ConnMaxLifetime,
			MaxConnIdleTime:   5 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
		}
		retryCfg := repository.DefaultRetryConfig()

		pgRepos, err := repository.NewPostgresRepositories(ctx, cfg.Database.DSN, poolCfg, retryCfg)
		if err != nil {
			return nil, fmt.Errorf("init postgres repositories: %w", err)
		}
		userRepo = pgRepos.Users
		sessionRepo = pgRepos.Sessions
		secretRepo = pgRepos.Secrets
		closer = pgRepos

		logger.Info("using PostgreSQL storage")
	} else {
		memRepos := repository.NewMemoryRepositories()
		userRepo = memRepos.Users
		sessionRepo = memRepos.Sessions
		secretRepo = memRepos.Secrets

		logger.Info("using in-memory storage (dev mode)")
	}

	// 2. Сервис аутентификации.
	authSvc := service.NewAuthService(
		userRepo,
		sessionRepo,
		cfg.Auth.JWTSecret,
		cfg.Auth.JWTExpiration,
		logger,
	)

	// 3. HTTP-хендлеры.
	authHandler := handler.NewAuthHandler(authSvc)

	// 4. Мидлварь аутентификации.
	authMw := middleware.AuthMiddleware(sessionRepo, cfg.Auth.JWTSecret)

	// 5. Роутер.
	router := chi.NewRouter()

	// Глобальные middleware.
	router.Use(chimw.RequestID)
	router.Use(chimw.Logger)
	router.Use(chimw.Recoverer)
	router.Use(chimw.Timeout(30 * time.Second))

	// Публичные роуты (без аутентификации).
	router.Post("/api/v1/register", authHandler.Register)
	router.Post("/api/v1/login", authHandler.Login)

	// Health check.
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Защищённые роуты (требуют JWT).
	router.Group(func(r chi.Router) {
		r.Use(authMw)

		secretHandler := handler.NewSecretHandler(secretRepo, logger)
		r.Post("/api/v1/secrets", secretHandler.Create)
		r.Get("/api/v1/secrets", secretHandler.List)
		r.Get("/api/v1/secrets/{id}", secretHandler.GetByID)
		r.Put("/api/v1/secrets/{id}", secretHandler.Update)
		r.Delete("/api/v1/secrets/{id}", secretHandler.Delete)
	})

	// 6. HTTP-сервер.
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{
		Server: srv,
		Logger: logger,
		closer: closer,
	}, nil
}

// Shutdown выполняет graceful shutdown сервера и закрывает репозитории.
func (a *App) Shutdown(ctx context.Context) error {
	if a.closer != nil {
		a.closer.Close()
	}
	return a.Server.Shutdown(ctx)
}
