package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"go-gophkeeper/internal/client/app"
)

func newRegisterCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "register [email]",
		Short: "Регистрация нового пользователя",
		Long: `Регистрирует нового пользователя в системе GophKeeper.
После успешной регистрации JWT токен сохраняется локально для последующих запросов.

Пример:
  gophkeeper register user@example.com`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]

			fmt.Fprint(os.Stderr, "Enter master password: ")
			password, err := readPassword()
			if err != nil {
				return fmt.Errorf("read password: %w", err)
			}

			fmt.Fprint(os.Stderr, "Confirm master password: ")
			confirm, err := readPassword()
			if err != nil {
				return fmt.Errorf("read confirm: %w", err)
			}

			if password != confirm {
				return fmt.Errorf("passwords do not match")
			}

			// Отправляем запрос на сервер.
			token, userID, err := a.Client.Register(email, password)
			if err != nil {
				return fmt.Errorf("register failed: %w", err)
			}

			// Сохраняем токен и UserID локально.
			if err := a.SaveToken(token); err != nil {
				return fmt.Errorf("save token: %w", err)
			}
			if err := a.SaveUserID(userID); err != nil {
				return fmt.Errorf("save user id: %w", err)
			}

			// Выводим ключ шифрования из мастер-пароля (ключ существует только в памяти).
			a.SetMasterKey(password, userID)

			fmt.Printf("\nRegistration successful!\n")
			fmt.Printf("   User ID: %s\n", userID)
			fmt.Printf("   Token saved to: %s\n", a.Config.TokenFile)

			return nil
		},
	}
}

// readPassword читает пароль из терминала без эха (безопасный ввод).
func readPassword() (string, error) {
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	fmt.Println() // перевод строки после ввода пароля
	return string(raw), nil
}
