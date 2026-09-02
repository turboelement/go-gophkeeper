// Package main — точка входа сервера GophKeeper.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"go-gophkeeper/internal/pkg/logger"
	"go-gophkeeper/internal/server/app"
	"go-gophkeeper/internal/server/config"
)

const shutdownTimeout = 30 * time.Second

func main() {
	// Загрузка конфигурации (env > флаги > JSON > defaults).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Инициализация логера с уровнем и режимом из конфига.
	zapLogger, err := logger.New(cfg.Logger.Level, cfg.Logger.DevMode)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync(zapLogger)

	zapLogger.Info("server starting",
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
		zap.String("log_level", cfg.Logger.Level),
		zap.Bool("dev_mode", cfg.Logger.DevMode),
	)

	// Сборка приложения (DI-контейнер).
	application, err := app.New(context.Background(), cfg, zapLogger)
	if err != nil {
		zapLogger.Fatal("failed to init application", zap.Error(err))
	}

	// Канал для сигналов ОС.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запуск HTTP-сервера в горутине.
	go func() {
		zapLogger.Info("server listening", zap.String("addr", application.Server.Addr))
		if err := application.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("server error", zap.Error(err))
		}
	}()

	// Ожидание сигнала завершения.
	sig := <-quit
	zapLogger.Info("shutting down server", zap.String("signal", sig.String()))

	// Graceful shutdown с таймаутом.
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		zapLogger.Fatal("graceful shutdown failed", zap.Error(err))
	}

	zapLogger.Info("server stopped")
}
