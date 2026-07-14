// Package config содержит конфигурацию CLI клиента GophKeeper.
package config

import (
	"os"
	"path/filepath"
)

// Config — конфигурация CLI клиента.
type Config struct {
	// ServerAddr — адрес сервера (host:port).
	ServerAddr string `json:"server_addr"`
	// TokenFile — путь к файлу с JWT токеном.
	TokenFile string `json:"token_file"`
	// DataDir — директория для хранения локального кеша.
	DataDir string `json:"data_dir"`
}

// Load загружает конфигурацию из env-переменных или использует значения по умолчанию.
func Load() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".gophkeeper")

	cfg := &Config{
		ServerAddr: getEnv("GOPHKEEPER_SERVER", "localhost:8080"),
		TokenFile:  filepath.Join(dataDir, "token"),
		DataDir:    dataDir,
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
