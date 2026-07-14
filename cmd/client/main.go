// Package main — точка входа клиента GophKeeper.
package main

import (
	"log"

	"go-gophkeeper/internal/client/app"
	"go-gophkeeper/internal/client/commands"
)

// Данные заполняются при сборке через ldflags:
//
//	go build -ldflags="-X main.version=1.0.0 -X main.buildDate=$(date -u +%Y-%m-%d) -X main.commitHash=$(git rev-parse --short HEAD)" ./cmd/client
var (
	version    = "dev"
	buildDate  = "unknown"
	commitHash = "none"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	defer application.Shutdown()

	commands.Execute(application, version, buildDate, commitHash)
}
