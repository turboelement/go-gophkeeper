package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
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

			fmt.Print("Enter master password: ")
			password, err := readPassword()
			if err != nil {
				return fmt.Errorf("read password: %w", err)
			}

			token, userID, err := gophkeeperApp.Client.Login(email, password)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			if err := gophkeeperApp.SaveToken(token); err != nil {
				return fmt.Errorf("save token: %w", err)
			}
			if err := gophkeeperApp.SaveUserID(userID); err != nil {
				return fmt.Errorf("save user id: %w", err)
			}

			fmt.Printf("\nLogin successful!\n")
			fmt.Printf("   User ID: %s\n", userID)

			return nil
		},
	}
}
