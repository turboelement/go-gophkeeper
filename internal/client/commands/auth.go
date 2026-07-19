package commands

import (
	"fmt"
	"os"

	"go-gophkeeper/internal/client/app"

	"golang.org/x/term"
)

// ensureStorageService гарантирует, что StorageService инициализирован.
// Если нет — загружает user_id из файла и запрашивает мастер-пароль.
func ensureStorageService(a *app.App) error {
	if a.StorageService != nil {
		return nil
	}

	// Пробуем загрузить токен, если ещё не загружен.
	if a.Client.GetToken() == "" {
		tokenData, err := os.ReadFile(a.Config.TokenFile)
		if err != nil || len(tokenData) == 0 {
			return fmt.Errorf("not authenticated. Please login first: gophkeeper login <email>")
		}
		a.Client.SetToken(string(tokenData))
	}

	// Читаем user_id из файла.
	uidData, err := os.ReadFile(a.Config.DataDir + "/user_id")
	if err != nil || len(uidData) == 0 {
		return fmt.Errorf("not authenticated. Please login first: gophkeeper login <email>")
	}
	userIDStr := string(uidData)

	// Запрашиваем мастер-пароль.
	fmt.Fprint(os.Stderr, "Enter master password: ")
	password, err := readPasswordStdin()
	if err != nil {
		return fmt.Errorf("read master password: %w", err)
	}

	a.SetMasterKey(password, userIDStr)

	if a.StorageService == nil {
		return fmt.Errorf("failed to initialize encryption. Please login again: gophkeeper login <email>")
	}

	return nil
}

// readPasswordStdin читает пароль из терминала без эха.
func readPasswordStdin() (string, error) {
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	fmt.Fprintln(os.Stderr)
	return string(raw), nil
}
