package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"go-gophkeeper/internal/client/app"
	httpclient "go-gophkeeper/internal/client/http"
	"go-gophkeeper/internal/crypto"
)

func newAddCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [credential|text|card|binary]",
		Short: "Добавить новый секрет",
		Long: `Добавляет новый секрет указанного типа.

Типы секретов:
  credential — логин/пароль
  text       — произвольный текст/заметка
  card       — банковская карта
  binary     — бинарные данные (base64)
Примеры:
  gophkeeper add credential --title "GitHub" --login "user" --password "pass"
  gophkeeper add text --title "Note" --content "Hello World"
  gophkeeper add card --title "Visa" --number "4111..." --holder "John" --cvv "123" --expires "12/28"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secretType := args[0]
			title, _ := cmd.Flags().GetString("title")
			if title == "" {
				return fmt.Errorf("title is required (--title)")
			}

			// Собираем payload в зависимости от типа.
			payload := make(map[string]string)
			switch secretType {
			case "credential":
				payload["login"], _ = cmd.Flags().GetString("login")
				payload["password"], _ = cmd.Flags().GetString("password")
				if payload["login"] == "" || payload["password"] == "" {
					return fmt.Errorf("credential requires --login and --password")
				}
			case "text":
				payload["content"], _ = cmd.Flags().GetString("content")
				if payload["content"] == "" {
					return fmt.Errorf("text requires --content")
				}
			case "card":
				payload["number"], _ = cmd.Flags().GetString("number")
				payload["holder"], _ = cmd.Flags().GetString("holder")
				payload["cvv"], _ = cmd.Flags().GetString("cvv")
				payload["expires"], _ = cmd.Flags().GetString("expires")
				if payload["number"] == "" || payload["holder"] == "" {
					return fmt.Errorf("card requires --number and --holder")
				}
			case "binary":
				payload["filename"], _ = cmd.Flags().GetString("filename")
				payload["content"], _ = cmd.Flags().GetString("content")
				if payload["filename"] == "" || payload["content"] == "" {
					return fmt.Errorf("binary requires --filename and --content")
				}
			default:
				return fmt.Errorf("unknown type: %s (use: credential, text, card, binary)", secretType)
			}

			// Шифруем payload.
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("marshal payload: %w", err)
			}

			// Используем ключ, полученный из мастер-пароля через Argon2id.
			if a.MasterKey == nil {
				return fmt.Errorf("not authenticated. Please login first: gophkeeper login <email>")
			}
			engine := crypto.NewAESGCMEngine(*a.MasterKey)

			encrypted, err := engine.Encrypt(payloadJSON)
			if err != nil {
				return fmt.Errorf("encrypt: %w", err)
			}

			req := httpclient.CreateSecretRequest{
				Type:             secretType,
				Title:            title,
				EncryptedPayload: encrypted,
			}

			if err := a.Client.CreateSecret(req); err != nil {
				return fmt.Errorf("create secret: %w", err)
			}

			fmt.Printf("\nSecret \"%s\" created successfully!\n", title)
			return nil
		},
	}

	// Общие флаги.
	cmd.Flags().StringP("title", "t", "", "название секрета")
	cmd.Flags().String("metadata", "", "метаданные (JSON)")

	// Флаги для credential.
	cmd.Flags().String("login", "", "логин")
	cmd.Flags().String("password", "", "пароль")

	// Флаги для text.
	cmd.Flags().String("content", "", "текстовое содержимое")

	// Флаги для card.
	cmd.Flags().String("number", "", "номер карты")
	cmd.Flags().String("holder", "", "держатель карты")
	cmd.Flags().String("cvv", "", "CVV код")
	cmd.Flags().String("expires", "", "срок действия (MM/YY)")

	// Флаги для binary.
	cmd.Flags().String("filename", "", "имя файла")

	return cmd
}
