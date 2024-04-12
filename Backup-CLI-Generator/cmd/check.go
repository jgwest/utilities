package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Output a diff between the expected script, and the actual script.",
	Long:  "Output a diff between the expected script, and the actual script.",
	Run: func(cmd *cobra.Command, args []string) {

		pathToConfigFile := args[0]
		scriptPath := args[1]

		backend := retrieveBackendFromConfigFile(pathToConfigFile)

		if !backend.SupportsBackupShellScriptDiffCheck() {
			reportCLIErrorAndExit(fmt.Errorf("backend '%v' does not support backup shell diff check", backend.ConfigType()))
			return
		}

		if err := backend.BackupShellScriptDiffCheck(pathToConfigFile, scriptPath); err != nil {
			reportCLIErrorAndExit(err)
			return
		}

	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Args = func(cmd *cobra.Command, args []string) error {

		if len(args) != 2 {
			return fmt.Errorf("two arguments required: (config file path) (shell script path)")
		}

		return nil
	}
}
