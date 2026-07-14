package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRegisterCmd() *cobra.Command {
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

			fmt.Print("Enter master password: ")
			password, err := readPassword()
			if err != nil {
				return fmt.Errorf("read password: %w", err)
			}

			fmt.Print("Confirm master password: ")
			confirm, err := readPassword()
			if err != nil {
				return fmt.Errorf("read confirm: %w", err)
			}

			if password != confirm {
				return fmt.Errorf("passwords do not match")
			}

			// Отправляем запрос на сервер.
			token, userID, err := gophkeeperApp.Client.Register(email, password)
			if err != nil {
				return fmt.Errorf("register failed: %w", err)
			}

			// Сохраняем токен и UserID локально.
			if err := gophkeeperApp.SaveToken(token); err != nil {
				return fmt.Errorf("save token: %w", err)
			}
			if err := gophkeeperApp.SaveUserID(userID); err != nil {
				return fmt.Errorf("save user id: %w", err)
			}
			fmt.Printf("\nRegistration successful!\n")
			fmt.Printf("   User ID: %s\n", userID)
			fmt.Printf("   Token saved to: %s\n", gophkeeperApp.Config.TokenFile)

			return nil
		},
	}
}

// readPassword читает пароль из терминала.
func readPassword() (string, error) {
	var pwd string
	_, err := fmt.Scanln(&pwd)
	return pwd, err
}
