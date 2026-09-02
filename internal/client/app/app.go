// Package app собирает зависимости клиента (конфиг, HTTP-клиент)
// и предоставляет единую точку входа для запуска CLI-приложения.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go-gophkeeper/internal/client/config"
	httpclient "go-gophkeeper/internal/client/http"
	"go-gophkeeper/internal/client/service"
	"go-gophkeeper/internal/crypto"
	"go-gophkeeper/internal/domain/models"
	"go-gophkeeper/internal/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// App — собранное приложение CLI-клиента.
type App struct {
	Config         *config.Config
	Client         *httpclient.Client
	MasterKey      *[crypto.KeySize]byte // ключ шифрования, полученный из мастер-пароля через Argon2id
	UserID         uuid.UUID
	StorageService *service.StorageService
	SyncService    *service.SyncService
	Logger         *zap.Logger
}

// New создаёт и собирает приложение из конфига.
func New() (*App, error) {
	cfg := config.Load()

	httpCli := httpclient.New("http://" + cfg.ServerAddr)

	zapLogger, err := logger.New("info", false)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	a := &App{
		Config: cfg,
		Client: httpCli,
		Logger: zapLogger,
	}

	// Загружаем сохранённый токен, если есть.
	if token, err := os.ReadFile(cfg.TokenFile); err == nil && len(token) > 0 {
		httpCli.SetToken(string(token))
	}

	// Загружаем сохранённый UserID, если есть.
	if uidData, err := os.ReadFile(cfg.DataDir + "/user_id"); err == nil && len(uidData) > 0 {
		if uid, err := uuid.Parse(string(uidData)); err == nil {
			a.UserID = uid
		}
	}

	// Инициализируем сервисы.
	if err := a.initServices(); err != nil {
		return nil, fmt.Errorf("init services: %w", err)
	}

	return a, nil
}

// initServices инициализирует StorageService и SyncService.
func (a *App) initServices() error {
	// Создаём локальный репозиторий.
	localRepo, err := service.NewLocalRepository(a.Config.DataDir)
	if err != nil {
		return fmt.Errorf("local repo: %w", err)
	}

	// Создаём StorageService.
	if a.MasterKey != nil {
		engine := crypto.NewAESGCMEngine(*a.MasterKey)
		a.StorageService = service.NewStorageService(localRepo, engine, a.UserID)
	}

	// SyncService будет создан при вызове InitSyncService (после установки MasterKey).
	_ = localRepo
	return nil
}

// SetMasterKey выводит ключ шифрования из мастер-пароля и UserID и сохраняет в памяти.
func (a *App) SetMasterKey(masterPassword string, userID string) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return // невалидный UserID — ключ останется nil
	}

	a.UserID = uid
	key := crypto.DeriveKey(masterPassword, uid)
	a.MasterKey = &key

	// Пересоздаём StorageService и SyncService с новым ключом.
	a.rebuildServices()
}

// rebuildServices пересоздаёт StorageService и SyncService после установки MasterKey.
func (a *App) rebuildServices() {
	if a.MasterKey == nil {
		return
	}

	engine := crypto.NewAESGCMEngine(*a.MasterKey)
	localRepo, err := service.NewLocalRepository(a.Config.DataDir)
	if err != nil {
		a.Logger.Warn("rebuild services: local repo", zap.Error(err))
		return
	}

	// HTTP-клиент как SecretRepository.
	remoteRepo := &httpClientSecretRepository{client: a.Client, userID: a.UserID}

	a.StorageService = service.NewStorageService(localRepo, engine, a.UserID)

	a.SyncService = service.NewSyncService(
		localRepo,
		remoteRepo,
		engine,
		a.UserID,
		a.Logger,
	)
}

// httpClientSecretRepository — адаптер, реализующий interfaces.SecretRepository
// через httpclient.Client.
// ВАЖНО: Сервер определяет пользователя из JWT-токена, а не из параметров запроса.
// Параметр userID в ListByUser и Delete используется для валидации на клиенте —
// если он передан, адаптер проверяет, что он соответствует ожидаемому.
type httpClientSecretRepository struct {
	client *httpclient.Client
	userID uuid.UUID
}

func (r *httpClientSecretRepository) Create(_ context.Context, secret *models.Secret) error {
	req := httpclient.CreateSecretRequest{
		Type:             string(secret.Type),
		Title:            secret.Title,
		Metadata:         string(secret.Metadata),
		EncryptedPayload: secret.EncryptedPayload,
	}
	return r.client.CreateSecret(req)
}

func (r *httpClientSecretRepository) GetByID(_ context.Context, id uuid.UUID) (*models.Secret, error) {
	data, err := r.client.GetSecret(id.String())
	if err != nil {
		return nil, err
	}
	// GetSecret возвращает уже распарсенное поле data (сырой JSON объекта секрета).
	var secret struct {
		ID               uuid.UUID       `json:"id"`
		UserID           uuid.UUID       `json:"user_id"`
		Type             string          `json:"type"`
		Title            string          `json:"title"`
		Metadata         json.RawMessage `json:"metadata"`
		EncryptedPayload []byte          `json:"encrypted_payload"`
		CreatedAt        time.Time       `json:"created_at"`
		UpdatedAt        time.Time       `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &secret); err != nil {
		return nil, fmt.Errorf("parse secret: %w", err)
	}
	return &models.Secret{
		ID:               secret.ID,
		UserID:           secret.UserID,
		Type:             models.SecretType(secret.Type),
		Title:            secret.Title,
		Metadata:         secret.Metadata,
		EncryptedPayload: secret.EncryptedPayload,
		CreatedAt:        secret.CreatedAt,
		UpdatedAt:        secret.UpdatedAt,
	}, nil
}

func (r *httpClientSecretRepository) ListByUser(_ context.Context, userID uuid.UUID) ([]models.SecretMeta, error) {
	// Проверяем, что запрашиваемый userID совпадает с ожидаемым (если установлен).
	// Сервер всё равно использует JWT для идентификации, но это ловит ошибки на клиенте.
	if r.userID != uuid.Nil && userID != uuid.Nil && userID != r.userID {
		return nil, fmt.Errorf("userID mismatch: requested %s, but authenticated as %s", userID, r.userID)
	}

	data, err := r.client.ListSecrets()
	if err != nil {
		return nil, err
	}
	// ListSecrets возвращает уже распарсенное поле data (сырой JSON массив).
	var remoteMetas []struct {
		ID        uuid.UUID `json:"id"`
		Type      string    `json:"type"`
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(data, &remoteMetas); err != nil {
		return nil, fmt.Errorf("parse metas: %w", err)
	}
	metas := make([]models.SecretMeta, 0, len(remoteMetas))
	for _, m := range remoteMetas {
		metas = append(metas, models.SecretMeta{
			ID:        m.ID,
			Type:      models.SecretType(m.Type),
			Title:     m.Title,
			CreatedAt: m.CreatedAt,
		})
	}
	return metas, nil
}

func (r *httpClientSecretRepository) Update(_ context.Context, secret *models.Secret) error {
	req := httpclient.CreateSecretRequest{
		Type:             string(secret.Type),
		Title:            secret.Title,
		Metadata:         string(secret.Metadata),
		EncryptedPayload: secret.EncryptedPayload,
	}
	return r.client.UpdateSecret(secret.ID.String(), req)
}

func (r *httpClientSecretRepository) Delete(_ context.Context, id uuid.UUID, userID uuid.UUID) error {
	// Проверяем, что удаление запрашивается от имени того же пользователя.
	if r.userID != uuid.Nil && userID != uuid.Nil && userID != r.userID {
		return fmt.Errorf("userID mismatch: requested %s, but authenticated as %s", userID, r.userID)
	}
	return r.client.DeleteSecret(id.String())
}

// WithServerAddr переопределяет адрес сервера (из флага --server).
func (a *App) WithServerAddr(addr string) *App {
	if addr != "" {
		a.Config.ServerAddr = addr
		a.Client = httpclient.New("http://" + addr)
		// Перезагружаем токен для нового адреса.
		if token, err := os.ReadFile(a.Config.TokenFile); err == nil && len(token) > 0 {
			a.Client.SetToken(string(token))
		}
		// Пересоздаём сервисы (remoteRepo изменился).
		a.rebuildServices()
	}
	return a
}

// SaveToken сохраняет JWT токен в файл и в клиент.
func (a *App) SaveToken(token string) error {
	a.Client.SetToken(token)
	return os.WriteFile(a.Config.TokenFile, []byte(token), 0600)
}

// SaveUserID сохраняет UserID в файл.
func (a *App) SaveUserID(userID string) error {
	userIDFile := a.Config.DataDir + "/user_id"
	return os.WriteFile(userIDFile, []byte(userID), 0600)
}

// Shutdown выполняет cleanup при завершении работы клиента.
func (a *App) Shutdown() {
	if a.Logger != nil {
		_ = a.Logger.Sync()
	}
}
