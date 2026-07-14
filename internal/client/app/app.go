// Package app собирает зависимости клиента (конфиг, HTTP-клиент)
// и предоставляет единую точку входа для запуска CLI-приложения.
package app

import (
	"os"

	"go-gophkeeper/internal/client/config"
	httpclient "go-gophkeeper/internal/client/http"
)

// App — собранное приложение CLI-клиента.
type App struct {
	Config *config.Config
	Client *httpclient.Client
}

// New создаёт и собирает приложение из конфига.
func New() (*App, error) {
	cfg := config.Load()

	httpCli := httpclient.New("http://" + cfg.ServerAddr)

	// Загружаем сохранённый токен, если есть.
	if token, err := os.ReadFile(cfg.TokenFile); err == nil && len(token) > 0 {
		httpCli.SetToken(string(token))
	}

	return &App{
		Config: cfg,
		Client: httpCli,
	}, nil
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
	// В будущем здесь можно добавить закрытие соединений,
	// сохранение кеша, сброс логов и т.д.
}
