// Package response предоставляет единый формат ответов API: {success, data, error, meta}.
package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse — универсальный ответ API.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// WriteJSON отправляет JSON-ответ с указанным HTTP-статусом.
func WriteJSON(w http.ResponseWriter, status int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// Success отправляет успешный ответ с данными.
func Success(w http.ResponseWriter, status int, data interface{}) {
	WriteJSON(w, status, APIResponse{
		Success: true,
		Data:    data,
	})
}

// Error отправляет ответ с ошибкой.
func Error(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, APIResponse{
		Success: false,
		Error:   msg,
	})
}
