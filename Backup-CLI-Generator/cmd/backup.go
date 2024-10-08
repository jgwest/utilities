package cmd

import (
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/model"
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			fmt.Println("args: (path to config file)")
			return
		}

		pathToConfigFile := args[0]

		model, err := model.ReadConfigFile(pathToConfigFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}

		backend, err := findBackendForConfigFile(model)
		if err != nil {
			fmt.Printf("unable to locate backend implementation for '%s'\n", pathToConfigFile)
			os.Exit(1)
			return
		}

		if err := backend.Backup(pathToConfigFile); err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}

	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

}
