package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		pathToConfigFile := getOptionalConfigFilePath(args)

		backend := retrieveBackendFromConfigFile(pathToConfigFile)

		if !backend.SupportsBackup() {
			reportCLIErrorAndExit(fmt.Errorf("backend '%v' does not support backup", backend.ConfigType()))
			return
		}

		if err := backend.Backup(pathToConfigFile); err != nil {
			reportCLIErrorAndExit(err)
			return
		}

	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

}
