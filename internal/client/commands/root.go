package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go-gophkeeper/internal/client/app"
)

var (
	gophkeeperApp *app.App
)

// RootCmd — корневая команда gophkeeper.
var RootCmd = &cobra.Command{
	Use:   "gophkeeper",
	Short: "GophKeeper — менеджер паролей и секретов",
	Long: `GophKeeper — клиент-серверное приложение для безопасного хранения
паролей, текстовых заметок, банковских карт и бинарных данных.
Все данные шифруются на стороне клиента (AES-256-GCM).
Сервер никогда не имеет доступа к незашифрованным данным.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Флаг --server переопределяет адрес сервера.
		if serverFlag, _ := cmd.Flags().GetString("server"); serverFlag != "" {
			gophkeeperApp.WithServerAddr(serverFlag)
		}
		return nil
	},
}

// Execute запускает CLI с переданным приложением.
func Execute(a *app.App, version, buildDate, commitHash string) {
	gophkeeperApp = a

	RootCmd.AddCommand(
		newRegisterCmd(),
		newLoginCmd(),
		newAddCmd(),
		newGetCmd(),
		newSyncCmd(),
		newVersionCmd(version, buildDate, commitHash),
	)

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringP("server", "s", "", "адрес сервера (host:port)")
}
