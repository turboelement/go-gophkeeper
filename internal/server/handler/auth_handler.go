// Package handler содержит HTTP-обработчики (транспортный слой).
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	domainerrors "go-gophkeeper/internal/domain/errors"
	"go-gophkeeper/internal/domain/interfaces"
	"go-gophkeeper/internal/pkg/response"
)

// AuthHandler обрабатывает запросы аутентификации.
type AuthHandler struct {
	authService interfaces.AuthService
}

// NewAuthHandler создаёт AuthHandler.
func NewAuthHandler(authService interfaces.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register обрабатывает POST /api/v1/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.authService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeError(w, err)
		return
	}

	response.Success(w, http.StatusCreated, map[string]string{
		"token": token,
	})
}

// Login обрабатывает POST /api/v1/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeError(w, err)
		return
	}

	response.Success(w, http.StatusOK, map[string]string{
		"token": token,
	})
}

func (h *AuthHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrInvalidCredentials):
		response.Error(w, http.StatusUnauthorized, "invalid email or password")
	case errors.Is(err, domainerrors.ErrAlreadyExists):
		response.Error(w, http.StatusConflict, "user already exists")
	default:
		response.Error(w, http.StatusBadRequest, err.Error())
	}
}
