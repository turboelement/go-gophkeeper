package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd(version, buildDate, commitHash string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать информацию о версии",
		Long: `Показывает версию клиента, дату сборки, коммит и информацию о платформе.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("GophKeeper CLI")
			fmt.Println("==============")
			fmt.Printf("Version:      %s\n", version)
			fmt.Printf("Build Date:   %s\n", buildDate)
			fmt.Printf("Commit Hash:  %s\n", commitHash)
			fmt.Printf("Go Version:   %s\n", runtime.Version())
			fmt.Printf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Println("==============")
		},
	}
}
