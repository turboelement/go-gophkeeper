package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go-gophkeeper/internal/client/app"
)

// Execute запускает CLI с переданным приложением.
func Execute(a *app.App, version, buildDate, commitHash string) {
	cmd := &cobra.Command{
		Use:   "gophkeeper",
		Short: "GophKeeper — менеджер паролей и секретов",
		Long: `GophKeeper — клиент-серверное приложение для безопасного хранения
паролей, текстовых заметок, банковских карт и бинарных данных.
Все данные шифруются на стороне клиента (AES-256-GCM).
Сервер никогда не имеет доступа к незашифрованным данным.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Флаг --server переопределяет адрес сервера.
			if serverFlag, _ := cmd.Flags().GetString("server"); serverFlag != "" {
				a.WithServerAddr(serverFlag)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringP("server", "s", "", "адрес сервера (host:port)")

	cmd.AddCommand(
		newRegisterCmd(a),
		newLoginCmd(a),
		newAddCmd(a),
		newGetCmd(a),
		newSyncCmd(a),
		newVersionCmd(version, buildDate, commitHash),
	)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
