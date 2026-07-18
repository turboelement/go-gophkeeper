package repository

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	domainerrors "go-gophkeeper/internal/domain/errors"
	"go-gophkeeper/internal/domain/interfaces"
	"go-gophkeeper/internal/domain/models"
)

// MemoryRepositories группирует все in-memory реализации репозиториев.
// Данные защищены единым sync.RWMutex, общим для всех суб-репозиториев.
type MemoryRepositories struct {
	mu       sync.RWMutex
	users    map[uuid.UUID]*models.User
	emails   map[string]uuid.UUID
	secrets  map[uuid.UUID]*models.Secret
	sessions map[string]*models.Session

	Users    interfaces.UserRepository
	Secrets  interfaces.SecretRepository
	Sessions interfaces.SessionRepository
}

// NewMemoryRepositories создаёт пустой MemoryRepositories, готовый к использованию.
func NewMemoryRepositories() *MemoryRepositories {
	r := &MemoryRepositories{
		users:    make(map[uuid.UUID]*models.User),
		emails:   make(map[string]uuid.UUID),
		secrets:  make(map[uuid.UUID]*models.Secret),
		sessions: make(map[string]*models.Session),
	}
	r.Users = &memoryUserRepository{parent: r}
	r.Secrets = &memorySecretRepository{parent: r}
	r.Sessions = &memorySessionRepository{parent: r}
	return r
}

// ----- memoryUserRepository -----

type memoryUserRepository struct {
	parent *MemoryRepositories
}

func (r *memoryUserRepository) Create(ctx context.Context, user *models.User) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	if _, exists := r.parent.emails[user.Email]; exists {
		return domainerrors.ErrAlreadyExists
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	r.parent.users[user.ID] = copyUser(user)
	r.parent.emails[user.Email] = user.ID
	return nil
}

// copyUser создаёт поверхностную копию User для предотвращения внешних мутаций.
func copyUser(u *models.User) *models.User {
	cp := *u
	return &cp
}

// copySecret создаёт копию Secret для предотвращения внешних мутаций.
func copySecret(s *models.Secret) *models.Secret {
	cp := *s
	return &cp
}

// copySession создаёт копию Session для предотвращения внешних мутаций.
func copySession(s *models.Session) *models.Session {
	cp := *s
	return &cp
}

func (r *memoryUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	r.parent.mu.RLock()
	defer r.parent.mu.RUnlock()

	user, ok := r.parent.users[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return copyUser(user), nil
}

func (r *memoryUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	r.parent.mu.RLock()
	defer r.parent.mu.RUnlock()

	id, ok := r.parent.emails[email]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	user, ok := r.parent.users[id]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	return copyUser(user), nil
}

func (r *memoryUserRepository) Update(ctx context.Context, user *models.User) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	existing, ok := r.parent.users[user.ID]
	if !ok {
		return domainerrors.ErrNotFound
	}

	oldEmail := existing.Email
	user.UpdatedAt = time.Now()

	if oldEmail != user.Email {
		delete(r.parent.emails, oldEmail)
		r.parent.emails[user.Email] = user.ID
	}

	r.parent.users[user.ID] = copyUser(user)
	return nil
}

func (r *memoryUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	user, ok := r.parent.users[id]
	if !ok {
		return domainerrors.ErrNotFound
	}

	delete(r.parent.emails, user.Email)
	delete(r.parent.users, id)
	return nil
}

// ----- memorySecretRepository -----

type memorySecretRepository struct {
	parent *MemoryRepositories
}

func (r *memorySecretRepository) Create(ctx context.Context, secret *models.Secret) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	now := time.Now()
	secret.CreatedAt = now
	secret.UpdatedAt = now

	r.parent.secrets[secret.ID] = secret
	return nil
}

func (r *memorySecretRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Secret, error) {
	r.parent.mu.RLock()
	defer r.parent.mu.RUnlock()

	secret, ok := r.parent.secrets[id]
	if !ok || secret.DeletedAt != nil {
		return nil, domainerrors.ErrNotFound
	}
	return copySecret(secret), nil
}

func (r *memorySecretRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.SecretMeta, error) {
	r.parent.mu.RLock()
	defer r.parent.mu.RUnlock()

	result := make([]models.SecretMeta, 0)
	for _, s := range r.parent.secrets {
		if s.UserID == userID && s.DeletedAt == nil {
			result = append(result, models.SecretMeta{
				ID:        s.ID,
				Type:      s.Type,
				Title:     s.Title,
				CreatedAt: s.CreatedAt,
			})
		}
	}
	return result, nil
}

func (r *memorySecretRepository) Update(ctx context.Context, secret *models.Secret) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	existing, ok := r.parent.secrets[secret.ID]
	if !ok || existing.DeletedAt != nil {
		return domainerrors.ErrNotFound
	}

	secret.UpdatedAt = time.Now()
	secret.CreatedAt = existing.CreatedAt
	r.parent.secrets[secret.ID] = secret
	return nil
}

func (r *memorySecretRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	secret, ok := r.parent.secrets[id]
	if !ok || secret.UserID != userID || secret.DeletedAt != nil {
		return domainerrors.ErrNotFound
	}

	now := time.Now()
	secret.DeletedAt = &now
	secret.UpdatedAt = now
	return nil
}

// ----- memorySessionRepository -----

type memorySessionRepository struct {
	parent *MemoryRepositories
}

func (r *memorySessionRepository) Create(ctx context.Context, session *models.Session) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	session.CreatedAt = time.Now()
	r.parent.sessions[session.Token] = session
	return nil
}

func (r *memorySessionRepository) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	r.parent.mu.RLock()
	defer r.parent.mu.RUnlock()

	session, ok := r.parent.sessions[token]
	if !ok || session.ExpiresAt.Before(time.Now()) {
		return nil, domainerrors.ErrNotFound
	}
	return copySession(session), nil
}

func (r *memorySessionRepository) Delete(ctx context.Context, token string) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	if _, ok := r.parent.sessions[token]; !ok {
		return domainerrors.ErrNotFound
	}
	delete(r.parent.sessions, token)
	return nil
}

func (r *memorySessionRepository) CleanupExpired(ctx context.Context, before time.Time) error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	for token, session := range r.parent.sessions {
		if session.ExpiresAt.Before(before) {
			delete(r.parent.sessions, token)
		}
	}
	return nil
}
