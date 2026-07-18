package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadFromArgs_Defaults проверяет, что без флагов и env
// LoadFromArgs возвращает конфиг со значениями по умолчанию
// (кроме обязательного JWTSecret, без которого валидация не пройдёт).
func TestLoadFromArgs_Defaults(t *testing.T) {
	// JWTSecret обязателен — передаём через флаг.
	args := []string{"--jwt-secret=test-secret"}
	cfg, err := LoadFromArgs(args)
	require.NoError(t, err)

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 10*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.Server.WriteTimeout)

	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 10, cfg.Database.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)

	assert.Equal(t, "test-secret", cfg.Auth.JWTSecret)
	assert.Equal(t, 24*time.Hour, cfg.Auth.JWTExpiration)

	assert.Equal(t, "info", cfg.Logger.Level)
	assert.Equal(t, false, cfg.Logger.DevMode)
}

// TestLoadFromArgs_Flags проверяет переопределение всех полей через флаги.
func TestLoadFromArgs_Flags(t *testing.T) {
	args := []string{
		"--server-host=127.0.0.1",
		"--server-port=9090",
		"--server-read-timeout=30s",
		"--server-write-timeout=15s",
		"--db-dsn=postgres://user:pass@localhost/mydb",
		"--db-max-open-conns=50",
		"--db-max-idle-conns=20",
		"--db-conn-max-lifetime=10m",
		"--jwt-secret=my-super-secret",
		"--jwt-expiration=48h",
		"--log-level=debug",
		"--log-dev-mode=true",
	}
	cfg, err := LoadFromArgs(args)
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 15*time.Second, cfg.Server.WriteTimeout)

	assert.Equal(t, "postgres://user:pass@localhost/mydb", cfg.Database.DSN)
	assert.Equal(t, 50, cfg.Database.MaxOpenConns)
	assert.Equal(t, 20, cfg.Database.MaxIdleConns)
	assert.Equal(t, 10*time.Minute, cfg.Database.ConnMaxLifetime)

	assert.Equal(t, "my-super-secret", cfg.Auth.JWTSecret)
	assert.Equal(t, 48*time.Hour, cfg.Auth.JWTExpiration)

	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, true, cfg.Logger.DevMode)
}

// TestLoadFromArgs_Env проверяет, что переменные окружения
// переопределяют флаги (наивысший приоритет).
func TestLoadFromArgs_Env(t *testing.T) {
	// Выставляем env.
	t.Setenv("SERVER_HOST", "10.0.0.1")
	t.Setenv("SERVER_PORT", "3000")
	t.Setenv("SERVER_READ_TIMEOUT", "5s")
	t.Setenv("SERVER_WRITE_TIMEOUT", "3s")
	t.Setenv("DATABASE_DSN", "postgres://env:only@localhost/db")
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "100")
	t.Setenv("DATABASE_MAX_IDLE_CONNS", "30")
	t.Setenv("DATABASE_CONN_MAX_LIFETIME", "1m")
	t.Setenv("JWT_SECRET", "env-secret")
	t.Setenv("JWT_EXPIRATION", "1h")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("LOG_DEV_MODE", "true")

	// Флаги задаём другие — env должен перезаписать их.
	args := []string{
		"--server-host=ignored",
		"--server-port=9999",
		"--jwt-secret=ignored",
		"--log-level=debug",
	}
	cfg, err := LoadFromArgs(args)
	require.NoError(t, err)

	assert.Equal(t, "10.0.0.1", cfg.Server.Host)
	assert.Equal(t, 3000, cfg.Server.Port)
	assert.Equal(t, 5*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 3*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, "postgres://env:only@localhost/db", cfg.Database.DSN)
	assert.Equal(t, 100, cfg.Database.MaxOpenConns)
	assert.Equal(t, 30, cfg.Database.MaxIdleConns)
	assert.Equal(t, 1*time.Minute, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, "env-secret", cfg.Auth.JWTSecret)
	assert.Equal(t, 1*time.Hour, cfg.Auth.JWTExpiration)
	assert.Equal(t, "error", cfg.Logger.Level)
	assert.Equal(t, true, cfg.Logger.DevMode)
}

// TestLoadFromArgs_JSONFile проверяет загрузку из JSON-файла и то,
// что флаги переопределяют JSON (флаги парсятся после JSON).
func TestLoadFromArgs_JSONFile(t *testing.T) {
	content := `{
		"server_host": "192.168.1.1",
		"server_port": 4444,
		"jwt_secret": "json-secret"
	}`
	tmpFile, err := os.CreateTemp(t.TempDir(), "config-*.json")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	t.Setenv("CONFIG_PATH", tmpFile.Name())
	cfg, err := LoadFromArgs([]string{})
	require.NoError(t, err)

	assert.Equal(t, "192.168.1.1", cfg.Server.Host)
	assert.Equal(t, 4444, cfg.Server.Port)
	assert.Equal(t, "json-secret", cfg.Auth.JWTSecret)

	t.Setenv("CONFIG_PATH", tmpFile.Name())
	cfg, err = LoadFromArgs([]string{"--server-host=10.10.10.10", "--jwt-secret=json-secret"})
	require.NoError(t, err)
	assert.Equal(t, "10.10.10.10", cfg.Server.Host)
}

// TestLoadFromArgs_ValidationError проверяет ошибку валидации без JWTSecret.
func TestLoadFromArgs_ValidationError(t *testing.T) {
	_, err := LoadFromArgs([]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET is required")
}

// TestLoadFromArgs_InvalidPort проверяет ошибку при невалидном порте.
func TestLoadFromArgs_InvalidPort(t *testing.T) {
	_, err := LoadFromArgs([]string{
		"--jwt-secret=test",
		"--server-port=0",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")

	_, err = LoadFromArgs([]string{
		"--jwt-secret=test",
		"--server-port=70000",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// TestLoadFromArgs_InvalidLogLevel проверяет ошибку при неверном уровне логирования.
func TestLoadFromArgs_InvalidLogLevel(t *testing.T) {
	_, err := LoadFromArgs([]string{
		"--jwt-secret=test",
		"--log-level=trace",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LOG_LEVEL")
}

// TestApplyJSONFile_InvalidPath проверяет ошибку при несуществующем JSON-файле.
func TestApplyJSONFile_InvalidPath(t *testing.T) {
	cfg := &Config{}
	err := cfg.applyJSONFile("/nonexistent/config.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read")
}

// TestApplyJSONFile_InvalidContent проверяет ошибку при невалидном JSON.
func TestApplyJSONFile_InvalidContent(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "bad-*.json")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("{bad json}")
	require.NoError(t, err)
	tmpFile.Close()

	cfg := &Config{}
	err = cfg.applyJSONFile(tmpFile.Name())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse")
}
