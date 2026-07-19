package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"go-gophkeeper/internal/client/app"
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
			if err := ensureStorageService(a); err != nil {
				return err
			}

			listMode, _ := cmd.Flags().GetBool("list")

			if listMode {
				return listSecrets(a)
			}

			if len(args) == 0 {
				return fmt.Errorf("secret ID is required (use --list to show all secrets)")
			}

			secretID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid secret ID: %w", err)
			}

			// Получаем расшифрованный секрет через StorageService.
			secretData, err := a.StorageService.GetByID(cmd.Context(), secretID)
			if err != nil {
				return fmt.Errorf("get secret: %w", err)
			}

			// Выводим секрет.
			fmt.Printf("\n=== Secret: %s ===\n", secretData.Title)
			fmt.Printf("  Type:  %s\n", secretData.Type)
			if secretData.Metadata != nil {
				if metaJSON, err := json.Marshal(secretData.Metadata); err == nil {
					fmt.Printf("  Metadata: %s\n", string(metaJSON))
				}
			}
			fmt.Println("--- Decrypted Payload ---")
			for k, v := range secretData.Payload {
				fmt.Printf("  %s: %s\n", k, v)
			}
			fmt.Println("-------------------------")

			return nil
		},
	}

	cmd.Flags().BoolP("list", "l", false, "показать список всех секретов")

	return cmd
}

// listSecrets получает список секретов через StorageService и выводит в виде таблицы.
func listSecrets(a *app.App) error {
	items, err := a.StorageService.List(context.Background())
	if err != nil {
		return fmt.Errorf("list secrets: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("No secrets found.")
		return nil
	}

	fmt.Printf("\nSecrets (%d total):\n", len(items))
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-38s %-12s %s\n", "ID", "Type", "Title")
	fmt.Println(strings.Repeat("-", 60))
	for _, s := range items {
		shortID := s.ID.String()
		if len(shortID) > 36 {
			shortID = shortID[:36]
		}
		fmt.Printf("%-38s %-12s %s\n", shortID, s.Type, s.Title)
	}
	fmt.Println(strings.Repeat("-", 60))

	return nil
}
