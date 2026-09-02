package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"go-gophkeeper/internal/client/app"
)

func newSyncCmd(a *app.App) *cobra.Command {
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

			if err := ensureStorageService(a); err != nil {
				return err
			}

			if a.SyncService == nil {
				return fmt.Errorf("sync service not initialized. Please login first")
			}

			ctx := context.Background()

			if !pushOnly {
				fmt.Println("Pulling changes from server...")
				count, err := a.SyncService.Pull(ctx)
				if err != nil {
					fmt.Printf("Pull warning: %v\n", err)
				} else {
					fmt.Printf("Pulled %d changes\n", count)
				}
			}

			if !pullOnly {
				fmt.Println("Pushing local changes...")
				count, err := a.SyncService.Push(ctx)
				if err != nil {
					fmt.Printf("Push warning: %v\n", err)
				} else {
					fmt.Printf("Pushed %d changes\n", count)
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
func pullChanges(a *app.App) error {
	ctx := context.Background()
	count, err := a.SyncService.Pull(ctx)
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	fmt.Printf("Pulled %d changes\n", count)
	return nil
}

// pushChanges отправляет локальные изменения на сервер.
func pushChanges(a *app.App) error {
	ctx := context.Background()
	count, err := a.SyncService.Push(ctx)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	fmt.Printf("Pushed %d changes\n", count)
	return nil
}
