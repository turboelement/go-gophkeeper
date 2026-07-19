// Package service содержит бизнес-логику (use cases) сервера.
package service

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"go-gophkeeper/internal/auth"
	domainerrors "go-gophkeeper/internal/domain/errors"
	"go-gophkeeper/internal/domain/interfaces"
	"go-gophkeeper/internal/domain/models"
)

// AuthService — реализация сервиса аутентификации.
type AuthService struct {
	userRepo    interfaces.UserRepository
	sessionRepo interfaces.SessionRepository
	jwtSecret   string
	jwtExpiry   time.Duration
	logger      *zap.Logger
}

// NewAuthService создаёт AuthService.
func NewAuthService(
	userRepo interfaces.UserRepository,
	sessionRepo interfaces.SessionRepository,
	jwtSecret string,
	jwtExpiry time.Duration,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		jwtSecret:   jwtSecret,
		jwtExpiry:   jwtExpiry,
		logger:      logger,
	}
}

// Register создаёт пользователя и возвращает JWT-токен и UUID пользователя.
func (s *AuthService) Register(ctx context.Context, email, password string) (string, uuid.UUID, error) {
	if err := validateEmail(email); err != nil {
		return "", uuid.Nil, fmt.Errorf("invalid email: %w", err)
	}
	if err := validatePassword(password); err != nil {
		return "", uuid.Nil, fmt.Errorf("invalid password: %w", err)
	}

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && err != domainerrors.ErrNotFound {
		return "", uuid.Nil, fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return "", uuid.Nil, domainerrors.ErrAlreadyExists
	}

	passwordHash, err := auth.Hash(password)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &models.User{
		ID:                 uuid.New(),
		Email:              email,
		MasterPasswordHash: passwordHash,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return "", uuid.Nil, fmt.Errorf("create user: %w", err)
	}

	token, err := auth.Generate(user.ID, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("generate token: %w", err)
	}

	session := &models.Session{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.jwtExpiry),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", uuid.Nil, fmt.Errorf("create session: %w", err)
	}

	s.logger.Info("user registered",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email),
	)

	return token, user.ID, nil
}

// Login аутентифицирует пользователя и возвращает JWT-токен и UUID пользователя.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, uuid.UUID, error) {
	if err := validateEmail(email); err != nil {
		return "", uuid.Nil, fmt.Errorf("invalid email: %w", err)
	}
	if password == "" {
		return "", uuid.Nil, fmt.Errorf("password is required")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			return "", uuid.Nil, domainerrors.ErrInvalidCredentials
		}
		return "", uuid.Nil, fmt.Errorf("get user: %w", err)
	}

	if err := auth.Verify(user.MasterPasswordHash, password); err != nil {
		return "", uuid.Nil, domainerrors.ErrInvalidCredentials
	}

	token, err := auth.Generate(user.ID, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("generate token: %w", err)
	}

	session := &models.Session{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.jwtExpiry),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", uuid.Nil, fmt.Errorf("create session: %w", err)
	}

	s.logger.Info("user logged in",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email),
	)

	return token, user.ID, nil
}

func validateEmail(email string) error {
	if email == "" || !utf8.ValidString(email) {
		return fmt.Errorf("email is required")
	}
	return nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}
