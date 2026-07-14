// Пакет config загружает конфигурацию сервера из JSON-файла, переменных окружения
// и флагов командной строки. Приоритет: env > флаги > JSON > defaults.
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config содержит всю конфигурацию сервера.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Logger   LoggerConfig
}

// ServerConfig — настройки HTTP-сервера.
type ServerConfig struct {
	Host         string        // Хост для прослушивания
	Port         int           // Порт HTTP-сервера
	ReadTimeout  time.Duration // Таймаут на чтение запроса
	WriteTimeout time.Duration // Таймаут на запись ответа
}

// DatabaseConfig — настройки подключения к PostgreSQL.
type DatabaseConfig struct {
	DSN             string        // PostgreSQL DSN (опциональный; если пустой — in-memory dev mode)
	MaxOpenConns    int           // Максимум открытых соединений
	MaxIdleConns    int           // Максимум idle-соединений
	ConnMaxLifetime time.Duration // Время жизни соединения
}

// AuthConfig — настройки аутентификации.
type AuthConfig struct {
	JWTSecret     string        // Секрет для подписи JWT (обязательный)
	JWTExpiration time.Duration // Время жизни JWT-токена
}

// LoggerConfig — настройки логирования.
type LoggerConfig struct {
	Level   string // Уровень логирования: debug, info, warn, error
	DevMode bool   // Режим разработки (консольный вывод)
}

const (
	envConfigPath          = "CONFIG_PATH"
	envServerHost          = "SERVER_HOST"
	envServerPort          = "SERVER_PORT"
	envServerReadTimeout   = "SERVER_READ_TIMEOUT"
	envServerWriteTimeout  = "SERVER_WRITE_TIMEOUT"
	envDatabaseDSN         = "DATABASE_DSN"
	envDatabaseMaxOpen     = "DATABASE_MAX_OPEN_CONNS"
	envDatabaseMaxIdle     = "DATABASE_MAX_IDLE_CONNS"
	envDatabaseMaxLifetime = "DATABASE_CONN_MAX_LIFETIME"
	envJWTSecret           = "JWT_SECRET"
	envJWTExpiration       = "JWT_EXPIRATION"
	envLogLevel            = "LOG_LEVEL"
	envLogDevMode          = "LOG_DEV_MODE"

	defaultServerHost      = "0.0.0.0"
	defaultServerPort      = 8080
	defaultReadTimeout     = 10 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 10
	defaultConnMaxLifetime = 5 * time.Minute
	defaultJWTExpiration   = 24 * time.Hour
	defaultLogLevel        = "info"
)

var (
	flagConfigPath          = flag.String("c", "", "Путь к JSON-файлу конфигурации")
	flagServerHost          = flag.String("server-host", "", "Хост для прослушивания")
	flagServerPort          = flag.Int("server-port", 0, "Порт HTTP-сервера")
	flagServerReadTimeout   = flag.String("server-read-timeout", "", "Таймаут на чтение запроса (например, 10s)")
	flagServerWriteTimeout  = flag.String("server-write-timeout", "", "Таймаут на запись ответа (например, 10s)")
	flagDatabaseDSN         = flag.String("db-dsn", "", "PostgreSQL DSN")
	flagDatabaseMaxOpen     = flag.Int("db-max-open-conns", 0, "Максимум открытых соединений с БД")
	flagDatabaseMaxIdle     = flag.Int("db-max-idle-conns", 0, "Максимум idle-соединений с БД")
	flagDatabaseMaxLifetime = flag.String("db-conn-max-lifetime", "", "Время жизни соединения с БД (например, 5m)")
	flagJWTSecret           = flag.String("jwt-secret", "", "Секрет для подписи JWT")
	flagJWTExpiration       = flag.String("jwt-expiration", "", "Время жизни JWT-токена (например, 24h)")
	flagLogLevel            = flag.String("log-level", "", "Уровень логирования (debug, info, warn, error)")
	flagLogDevMode          = flag.Bool("log-dev-mode", false, "Режим разработки (цветной вывод)")
)

// jsonFile — структура для парсинга JSON-конфига с указателями
// для определения, было ли поле указано в файле.
type jsonFile struct {
	ServerHost          *string `json:"server_host"`
	ServerPort          *int    `json:"server_port"`
	ServerReadTimeout   *string `json:"server_read_timeout"`
	ServerWriteTimeout  *string `json:"server_write_timeout"`
	DatabaseDSN         *string `json:"database_dsn"`
	DatabaseMaxOpen     *int    `json:"database_max_open_conns"`
	DatabaseMaxIdle     *int    `json:"database_max_idle_conns"`
	DatabaseMaxLifetime *string `json:"database_conn_max_lifetime"`
	JWTSecret           *string `json:"jwt_secret"`
	JWTExpiration       *string `json:"jwt_expiration"`
	LogLevel            *string `json:"log_level"`
	LogDevMode          *bool   `json:"log_dev_mode"`
}

// Load загружает конфигурацию из JSON-файла, env и флагов, затем валидирует её.
func Load() (*Config, error) {
	if !flag.Parsed() {
		flag.Parse()
	}

	// Отслеживаем флаги, которые были явно установлены пользователем.
	visitedFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		visitedFlags[f.Name] = true
	})

	cfg := defaultConfig()

	// 1. Применить JSON-файл (самый низкий приоритет после defaults).
	if path := resolveConfigPath(visitedFlags); path != "" {
		if err := cfg.applyJSONFile(path); err != nil {
			return nil, fmt.Errorf("config file: %w", err)
		}
	}

	// 2. Применить явно переданные флаги (перезаписывают JSON).
	cfg.applyFlags(visitedFlags)

	// 3. Применить переменные окружения (перезаписывают флаги).
	cfg.applyEnv()

	// 4. Валидация обязательных полей.
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         defaultServerHost,
			Port:         defaultServerPort,
			ReadTimeout:  defaultReadTimeout,
			WriteTimeout: defaultWriteTimeout,
		},
		Database: DatabaseConfig{
			MaxOpenConns:    defaultMaxOpenConns,
			MaxIdleConns:    defaultMaxIdleConns,
			ConnMaxLifetime: defaultConnMaxLifetime,
		},
		Auth: AuthConfig{
			JWTExpiration: defaultJWTExpiration,
		},
		Logger: LoggerConfig{
			Level: defaultLogLevel,
		},
	}
}

func resolveConfigPath(visitedFlags map[string]bool) string {
	if visitedFlags["c"] {
		return *flagConfigPath
	}
	if v, ok := os.LookupEnv(envConfigPath); ok {
		return v
	}
	return ""
}

func (cfg *Config) applyJSONFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %q: %w", path, err)
	}

	var jf jsonFile
	if err := json.Unmarshal(data, &jf); err != nil {
		return fmt.Errorf("cannot parse %q: %w", path, err)
	}

	if jf.ServerHost != nil {
		cfg.Server.Host = *jf.ServerHost
	}
	if jf.ServerPort != nil {
		cfg.Server.Port = *jf.ServerPort
	}
	if jf.ServerReadTimeout != nil {
		if d, err := time.ParseDuration(*jf.ServerReadTimeout); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if jf.ServerWriteTimeout != nil {
		if d, err := time.ParseDuration(*jf.ServerWriteTimeout); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}
	if jf.DatabaseDSN != nil {
		cfg.Database.DSN = *jf.DatabaseDSN
	}
	if jf.DatabaseMaxOpen != nil {
		cfg.Database.MaxOpenConns = *jf.DatabaseMaxOpen
	}
	if jf.DatabaseMaxIdle != nil {
		cfg.Database.MaxIdleConns = *jf.DatabaseMaxIdle
	}
	if jf.DatabaseMaxLifetime != nil {
		if d, err := time.ParseDuration(*jf.DatabaseMaxLifetime); err == nil {
			cfg.Database.ConnMaxLifetime = d
		}
	}
	if jf.JWTSecret != nil {
		cfg.Auth.JWTSecret = *jf.JWTSecret
	}
	if jf.JWTExpiration != nil {
		if d, err := time.ParseDuration(*jf.JWTExpiration); err == nil {
			cfg.Auth.JWTExpiration = d
		}
	}
	if jf.LogLevel != nil {
		cfg.Logger.Level = *jf.LogLevel
	}
	if jf.LogDevMode != nil {
		cfg.Logger.DevMode = *jf.LogDevMode
	}

	return nil
}

func (cfg *Config) applyFlags(visitedFlags map[string]bool) {
	if visitedFlags["server-host"] {
		cfg.Server.Host = *flagServerHost
	}
	if visitedFlags["server-port"] {
		cfg.Server.Port = *flagServerPort
	}
	if visitedFlags["server-read-timeout"] {
		if d, err := time.ParseDuration(*flagServerReadTimeout); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if visitedFlags["server-write-timeout"] {
		if d, err := time.ParseDuration(*flagServerWriteTimeout); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}
	if visitedFlags["db-dsn"] {
		cfg.Database.DSN = *flagDatabaseDSN
	}
	if visitedFlags["db-max-open-conns"] {
		cfg.Database.MaxOpenConns = *flagDatabaseMaxOpen
	}
	if visitedFlags["db-max-idle-conns"] {
		cfg.Database.MaxIdleConns = *flagDatabaseMaxIdle
	}
	if visitedFlags["db-conn-max-lifetime"] {
		if d, err := time.ParseDuration(*flagDatabaseMaxLifetime); err == nil {
			cfg.Database.ConnMaxLifetime = d
		}
	}
	if visitedFlags["jwt-secret"] {
		cfg.Auth.JWTSecret = *flagJWTSecret
	}
	if visitedFlags["jwt-expiration"] {
		if d, err := time.ParseDuration(*flagJWTExpiration); err == nil {
			cfg.Auth.JWTExpiration = d
		}
	}
	if visitedFlags["log-level"] {
		cfg.Logger.Level = *flagLogLevel
	}
	if visitedFlags["log-dev-mode"] {
		cfg.Logger.DevMode = *flagLogDevMode
	}
}

func (cfg *Config) applyEnv() {
	if v, ok := os.LookupEnv(envServerHost); ok {
		cfg.Server.Host = v
	}
	if v, ok := os.LookupEnv(envServerPort); ok {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v, ok := os.LookupEnv(envServerReadTimeout); ok {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if v, ok := os.LookupEnv(envServerWriteTimeout); ok {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}
	if v, ok := os.LookupEnv(envDatabaseDSN); ok {
		cfg.Database.DSN = v
	}
	if v, ok := os.LookupEnv(envDatabaseMaxOpen); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxOpenConns = n
		}
	}
	if v, ok := os.LookupEnv(envDatabaseMaxIdle); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxIdleConns = n
		}
	}
	if v, ok := os.LookupEnv(envDatabaseMaxLifetime); ok {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Database.ConnMaxLifetime = d
		}
	}
	if v, ok := os.LookupEnv(envJWTSecret); ok {
		cfg.Auth.JWTSecret = v
	}
	if v, ok := os.LookupEnv(envJWTExpiration); ok {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.JWTExpiration = d
		}
	}
	if v, ok := os.LookupEnv(envLogLevel); ok {
		cfg.Logger.Level = v
	}
	if v, ok := os.LookupEnv(envLogDevMode); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Logger.DevMode = b
		}
	}
}

func (cfg *Config) validate() error {
	if cfg.Auth.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("SERVER_PORT %d is out of range 1–65535", cfg.Server.Port)
	}
	if cfg.Logger.Level != "debug" && cfg.Logger.Level != "info" && cfg.Logger.Level != "warn" && cfg.Logger.Level != "error" {
		return fmt.Errorf("LOG_LEVEL %q must be one of: debug, info, warn, error", cfg.Logger.Level)
	}

	return nil
}
