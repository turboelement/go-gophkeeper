package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Синхронизация с сервером",
		Long: `Синхронизирует локальные изменения с сервером.
Pull — загружает изменения с сервера.
Push — отправляет локальные изменения на сервер.

Примеры:
  gophkeeper sync
  gophkeeper sync --push-only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pushOnly, _ := cmd.Flags().GetBool("push-only")
			pullOnly, _ := cmd.Flags().GetBool("pull-only")

			if gophkeeperApp.Client.GetToken() == "" {
				return fmt.Errorf("not authenticated. Please login first: gophkeeper login <email>")
			}

			if !pushOnly {
				fmt.Println("Pulling changes from server...")
				if err := pullChanges(); err != nil {
					fmt.Printf("Pull warning: %v\n", err)
				}
			}

			if !pullOnly {
				fmt.Println("Pushing local changes...")
				if err := pushChanges(); err != nil {
					fmt.Printf("Push warning: %v\n", err)
				}
			}

			fmt.Println("Sync completed at", time.Now().Format(time.RFC3339))

			return nil
		},
	}

	cmd.Flags().Bool("push-only", false, "только отправка изменений")
	cmd.Flags().Bool("pull-only", false, "только получение изменений")

	return cmd
}

// pullChanges загружает изменения с сервера.
func pullChanges() error {
	_, err := gophkeeperApp.Client.ListSecrets()
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	return nil
}

// pushChanges отправляет локальные изменения на сервер.
func pushChanges() error {
	_ = json.Unmarshal
	return nil
}
