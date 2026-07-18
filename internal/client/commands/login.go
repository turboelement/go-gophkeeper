package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go-gophkeeper/internal/client/app"
)

func newLoginCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "login [email]",
		Short: "Аутентификация пользователя",
		Long: `Выполняет вход в систему GophKeeper.
JWT токен сохраняется локально для последующих запросов.

Пример:
  gophkeeper login user@example.com`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]

			fmt.Fprint(os.Stderr, "Enter master password: ")
			password, err := readPassword()
			if err != nil {
				return fmt.Errorf("read password: %w", err)
			}

			token, userID, err := a.Client.Login(email, password)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			if err := a.SaveToken(token); err != nil {
				return fmt.Errorf("save token: %w", err)
			}
			if err := a.SaveUserID(userID); err != nil {
				return fmt.Errorf("save user id: %w", err)
			}

			// Выводим ключ шифрования из мастер-пароля (ключ существует только в памяти).
			a.SetMasterKey(password, userID)

			fmt.Printf("\nLogin successful!\n")
			fmt.Printf("   User ID: %s\n", userID)

			return nil
		},
	}
}
