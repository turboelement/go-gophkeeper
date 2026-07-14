package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New создаёт *zap.Logger с настраиваемым уровнем и режимом вывода.
// level — пороговый уровень логирования: "debug", "info", "warn", "error".
// devMode — true: человекочитаемый консольный вывод; false: структурированный JSON.
func New(level string, devMode bool) (*zap.Logger, error) {
	var cfg zap.Config

	if devMode {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	cfg.Level = zap.AtomicLevel{}
	if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}

	return cfg.Build()
}

// Sync принудительно сбрасывает буфер логера.
// Должен вызываться перед завершением приложения: defer logger.Sync(log).
func Sync(logger *zap.Logger) {
	_ = logger.Sync()
}
