// Package config загружает конфигурацию сервера из JSON-файла, переменных окружения
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

// Load загружает конфигурацию из os.Args и переменных окружения.
func Load() (*Config, error) {
	return LoadFromArgs(os.Args[1:])
}

// LoadFromArgs загружает конфигурацию из переданного списка аргументов и env.
//
// Параметр args не должен включать имя программы (os.Args[0]).
func LoadFromArgs(args []string) (*Config, error) {
	// 1. Defaults.
	cfg := &Config{
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

	// 2. JSON-файл, если путь указан в env.
	if path := os.Getenv(envConfigPath); path != "" {
		if err := cfg.applyJSONFile(path); err != nil {
			return nil, fmt.Errorf("config file: %w", err)
		}
	}

	// 3. Флаги командной строки (перезаписывают JSON).
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	configPath := flags.String("c", "", "Путь к JSON-файлу конфигурации")
	serverHost := flags.String("server-host", cfg.Server.Host, "Хост для прослушивания")
	serverPort := flags.Int("server-port", cfg.Server.Port, "Порт HTTP-сервера")
	serverReadTimeout := flags.String("server-read-timeout", cfg.Server.ReadTimeout.String(), "Таймаут на чтение запроса (например, 10s)")
	serverWriteTimeout := flags.String("server-write-timeout", cfg.Server.WriteTimeout.String(), "Таймаут на запись ответа (например, 10s)")
	databaseDSN := flags.String("db-dsn", cfg.Database.DSN, "PostgreSQL DSN")
	databaseMaxOpen := flags.Int("db-max-open-conns", cfg.Database.MaxOpenConns, "Максимум открытых соединений с БД")
	databaseMaxIdle := flags.Int("db-max-idle-conns", cfg.Database.MaxIdleConns, "Максимум idle-соединений с БД")
	databaseMaxLifetime := flags.String("db-conn-max-lifetime", cfg.Database.ConnMaxLifetime.String(), "Время жизни соединения с БД (например, 5m)")
	jwtSecret := flags.String("jwt-secret", cfg.Auth.JWTSecret, "Секрет для подписи JWT")
	jwtExpiration := flags.String("jwt-expiration", cfg.Auth.JWTExpiration.String(), "Время жизни JWT-токена (например, 24h)")
	logLevel := flags.String("log-level", cfg.Logger.Level, "Уровень логирования (debug, info, warn, error)")
	logDevMode := flags.Bool("log-dev-mode", cfg.Logger.DevMode, "Режим разработки (цветной вывод)")

	if err := flags.Parse(args); err != nil {
		return nil, err
	}

	// Применяем значения из флагов, если они изменились относительно defaults.
	// flag.NewFlagSet сам подставляет defaults, поэтому проверяем через Visit.
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "c":
			if *configPath != "" {
				if err := cfg.applyJSONFile(*configPath); err != nil {
					return // ошибка сохранится при валидации
				}
			}
		case "server-host":
			cfg.Server.Host = *serverHost
		case "server-port":
			cfg.Server.Port = *serverPort
		case "server-read-timeout":
			if d, err := time.ParseDuration(*serverReadTimeout); err == nil {
				cfg.Server.ReadTimeout = d
			}
		case "server-write-timeout":
			if d, err := time.ParseDuration(*serverWriteTimeout); err == nil {
				cfg.Server.WriteTimeout = d
			}
		case "db-dsn":
			cfg.Database.DSN = *databaseDSN
		case "db-max-open-conns":
			cfg.Database.MaxOpenConns = *databaseMaxOpen
		case "db-max-idle-conns":
			cfg.Database.MaxIdleConns = *databaseMaxIdle
		case "db-conn-max-lifetime":
			if d, err := time.ParseDuration(*databaseMaxLifetime); err == nil {
				cfg.Database.ConnMaxLifetime = d
			}
		case "jwt-secret":
			cfg.Auth.JWTSecret = *jwtSecret
		case "jwt-expiration":
			if d, err := time.ParseDuration(*jwtExpiration); err == nil {
				cfg.Auth.JWTExpiration = d
			}
		case "log-level":
			cfg.Logger.Level = *logLevel
		case "log-dev-mode":
			cfg.Logger.DevMode = *logDevMode
		}
	})

	// 4. Переменные окружения (наивысший приоритет).
	cfg.applyEnv()

	// 5. Валидация.
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
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
