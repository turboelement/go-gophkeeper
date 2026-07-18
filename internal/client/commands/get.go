package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"go-gophkeeper/internal/client/app"
	"go-gophkeeper/internal/crypto"
)

func newGetCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Получить секрет по ID",
		Long: `Получает расшифрованный секрет по его идентификатору.

Примеры:
  gophkeeper get <secret-uuid>
  gophkeeper get --list`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			listMode, _ := cmd.Flags().GetBool("list")

			if listMode {
				return listSecrets(a)
			}

			if len(args) == 0 {
				return fmt.Errorf("secret ID is required (use --list to show all secrets)")
			}

			secretID := args[0]

			// Получаем зашифрованные данные с сервера.

			data, err := a.Client.GetSecret(secretID)
			if err != nil {
				return fmt.Errorf("get secret: %w", err)
			}

			// Парсим ответ.
			var secret struct {
				ID               string          `json:"id"`
				Type             string          `json:"type"`
				Title            string          `json:"title"`
				Metadata         json.RawMessage `json:"metadata"`
				EncryptedPayload []byte          `json:"encrypted_payload"`
				CreatedAt        string          `json:"created_at"`
			}
			if err := json.Unmarshal(data, &secret); err != nil {
				return fmt.Errorf("parse secret: %w", err)
			}

			// Используем ключ, полученный из мастер-пароля через Argon2id.
			if a.MasterKey == nil {
				return fmt.Errorf("not authenticated. Please login first: gophkeeper login <email>")
			}
			engine := crypto.NewAESGCMEngine(*a.MasterKey)

			plaintext, err := engine.Decrypt(secret.EncryptedPayload)
			if err != nil {
				return fmt.Errorf("decrypt: %w", err)
			}

			var payload map[string]string
			if err := json.Unmarshal(plaintext, &payload); err != nil {
				return fmt.Errorf("parse payload: %w", err)
			}

			// Выводим секрет.
			fmt.Printf("\n=== Secret: %s ===\n", secret.Title)
			fmt.Printf("  ID:    %s\n", secret.ID)
			fmt.Printf("  Type:  %s\n", secret.Type)
			fmt.Printf("  Created: %s\n", secret.CreatedAt)
			fmt.Println("--- Decrypted Payload ---")
			for k, v := range payload {
				fmt.Printf("  %s: %s\n", k, v)
			}
			fmt.Println("-------------------------")

			return nil
		},
	}

	cmd.Flags().BoolP("list", "l", false, "показать список всех секретов")

	return cmd
}

// listSecrets получает список секретов с сервера и выводит в виде таблицы.
func listSecrets(a *app.App) error {
	data, err := a.Client.ListSecrets()
	if err != nil {
		return fmt.Errorf("list secrets: %w", err)
	}

	var secrets []struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Title     string `json:"title"`
		CreatedAt string `json:"created_at"`
	}

	if err := json.Unmarshal(data, &secrets); err != nil {
		return fmt.Errorf("parse secrets list: %w", err)
	}

	if len(secrets) == 0 {
		fmt.Println("No secrets found.")
		return nil
	}

	fmt.Printf("\nSecrets (%d total):\n", len(secrets))
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-38s %-12s %s\n", "ID", "Type", "Title")
	fmt.Println(strings.Repeat("-", 60))
	for _, s := range secrets {
		shortID := s.ID
		if len(shortID) > 36 {
			shortID = shortID[:36]
		}
		fmt.Printf("%-38s %-12s %s\n", shortID, s.Type, s.Title)
	}
	fmt.Println(strings.Repeat("-", 60))

	return nil
}
