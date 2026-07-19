package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"go-gophkeeper/internal/client/app"
	"go-gophkeeper/internal/domain/interfaces"
	"go-gophkeeper/internal/domain/models"
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
			if err := ensureStorageService(a); err != nil {
				return err
			}

			secretType := args[0]
			title, _ := cmd.Flags().GetString("title")
			if title == "" {
				return fmt.Errorf("title is required (--title)")
			}

			// Нормализуем тип.
			var st models.SecretType
			switch secretType {
			case "credential":
				st = models.SecretCredential
			case "text":
				st = models.SecretText
			case "card":
				st = models.SecretCard
			case "binary":
				st = models.SecretBinary
			default:
				return fmt.Errorf("unknown type: %s (use: credential, text, card, binary)", secretType)
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
			}

			// Парсим metadata, если указан.
			var metadata map[string]string
			if metaStr, _ := cmd.Flags().GetString("metadata"); metaStr != "" {
				metadata = map[string]string{"raw": metaStr}
			}

			// Используем StorageService.Create — он сам зашифрует и сохранит локально.
			data := interfaces.SecretData{
				Type:     st,
				Title:    title,
				Metadata: metadata,
				Payload:  payload,
			}

			secret, err := a.StorageService.Create(cmd.Context(), data)
			if err != nil {
				return fmt.Errorf("create secret: %w", err)
			}

			fmt.Printf("\nSecret \"%s\" created successfully!\n", secret.Title)
			fmt.Printf("  ID: %s\n", secret.ID)
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
