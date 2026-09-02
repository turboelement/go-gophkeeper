// Package handler содержит HTTP-обработчики (транспортный слой).
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	domainerrors "go-gophkeeper/internal/domain/errors"
	"go-gophkeeper/internal/domain/interfaces"
	"go-gophkeeper/internal/domain/models"
	"go-gophkeeper/internal/pkg/response"
	"go-gophkeeper/internal/server/middleware"
)

// SecretHandler обрабатывает CRUD-запросы для секретов.
type SecretHandler struct {
	secretRepo interfaces.SecretRepository
	logger     *zap.Logger
}

// NewSecretHandler создаёт SecretHandler.
func NewSecretHandler(secretRepo interfaces.SecretRepository, logger *zap.Logger) *SecretHandler {
	return &SecretHandler{
		secretRepo: secretRepo,
		logger:     logger,
	}
}

type createSecretRequest struct {
	Type             string          `json:"type"`
	Title            string          `json:"title"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	EncryptedPayload []byte          `json:"encrypted_payload"`
}

type updateSecretRequest struct {
	Type             string          `json:"type,omitempty"`
	Title            string          `json:"title,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	EncryptedPayload []byte          `json:"encrypted_payload,omitempty"`
}

type secretResponse struct {
	ID               string          `json:"id"`
	UserID           string          `json:"user_id"`
	Type             string          `json:"type"`
	Title            string          `json:"title"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	EncryptedPayload []byte          `json:"encrypted_payload,omitempty"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
}

type secretMetaResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

// Create обрабатывает POST /api/v1/secrets.
func (h *SecretHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req createSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" || req.Type == "" || len(req.EncryptedPayload) == 0 {
		response.Error(w, http.StatusBadRequest, "title, type and encrypted_payload are required")
		return
	}

	if !isValidSecretType(req.Type) {
		response.Error(w, http.StatusBadRequest, "invalid secret type, must be one of: credential, text, card, binary")
		return
	}

	now := timeNow()
	secret := &models.Secret{
		ID:               uuid.New(),
		UserID:           userID,
		Type:             models.SecretType(req.Type),
		Title:            req.Title,
		Metadata:         req.Metadata,
		EncryptedPayload: req.EncryptedPayload,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := h.secretRepo.Create(r.Context(), secret); err != nil {
		h.logger.Error("failed to create secret", zap.Error(err))
		response.Error(w, http.StatusInternalServerError, "failed to create secret")
		return
	}

	response.Success(w, http.StatusCreated, toSecretResponse(secret))
}

// List обрабатывает GET /api/v1/secrets.
func (h *SecretHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	metas, err := h.secretRepo.ListByUser(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list secrets", zap.Error(err))
		response.Error(w, http.StatusInternalServerError, "failed to list secrets")
		return
	}

	resp := make([]secretMetaResponse, 0, len(metas))
	for _, m := range metas {
		resp = append(resp, secretMetaResponse{
			ID:        m.ID.String(),
			Type:      string(m.Type),
			Title:     m.Title,
			CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	response.Success(w, http.StatusOK, resp)
}

// GetByID обрабатывает GET /api/v1/secrets/{id}.
func (h *SecretHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid secret id")
		return
	}

	secret, err := h.secretRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainerrors.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "secret not found")
			return
		}
		h.logger.Error("failed to get secret", zap.Error(err))
		response.Error(w, http.StatusInternalServerError, "failed to get secret")
		return
	}

	if secret.UserID != userID {
		response.Error(w, http.StatusNotFound, "secret not found")
		return
	}

	response.Success(w, http.StatusOK, toSecretResponse(secret))
}

// Update обрабатывает PUT /api/v1/secrets/{id}.
func (h *SecretHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid secret id")
		return
	}

	// Достаём существующий секрет для проверки владельца.
	existing, err := h.secretRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domainerrors.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "secret not found")
			return
		}
		h.logger.Error("failed to get secret", zap.Error(err))
		response.Error(w, http.StatusInternalServerError, "failed to update secret")
		return
	}

	if existing.UserID != userID {
		response.Error(w, http.StatusNotFound, "secret not found")
		return
	}

	var req updateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Обновляем только переданные поля.
	if req.Type != "" {
		if !isValidSecretType(req.Type) {
			response.Error(w, http.StatusBadRequest, "invalid secret type")
			return
		}
		existing.Type = models.SecretType(req.Type)
	}
	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Metadata != nil {
		existing.Metadata = req.Metadata
	}
	if req.EncryptedPayload != nil {
		existing.EncryptedPayload = req.EncryptedPayload
	}
	existing.UpdatedAt = timeNow()

	if err := h.secretRepo.Update(r.Context(), existing); err != nil {
		h.logger.Error("failed to update secret", zap.Error(err))
		response.Error(w, http.StatusInternalServerError, "failed to update secret")
		return
	}

	response.Success(w, http.StatusOK, toSecretResponse(existing))
}

// Delete обрабатывает DELETE /api/v1/secrets/{id}.
func (h *SecretHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid secret id")
		return
	}

	if err := h.secretRepo.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, domainerrors.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "secret not found")
			return
		}
		h.logger.Error("failed to delete secret", zap.Error(err))
		response.Error(w, http.StatusInternalServerError, "failed to delete secret")
		return
	}

	response.Success(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Вспомогательные функции.

func isValidSecretType(t string) bool {
	switch t {
	case string(models.SecretCredential), string(models.SecretText),
		string(models.SecretCard), string(models.SecretBinary):
		return true
	}
	return false
}

func toSecretResponse(s *models.Secret) secretResponse {
	resp := secretResponse{
		ID:               s.ID.String(),
		UserID:           s.UserID.String(),
		Type:             string(s.Type),
		Title:            s.Title,
		Metadata:         s.Metadata,
		EncryptedPayload: s.EncryptedPayload,
		CreatedAt:        s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if resp.Metadata == nil {
		resp.Metadata = json.RawMessage("{}")
	}
	return resp
}

// timeNow — переопределяемая функция для тестов.
var timeNow = func() time.Time {
	return time.Now()
}
