package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// quickCheckCmd represents the check command
var quickCheckCmd = &cobra.Command{
	Use:   "quick-check",
	Short: "...",
	Long:  "...",
	Run: func(cmd *cobra.Command, args []string) {

		configFile := getOptionalConfigFilePath(args)

		backend := retrieveBackendFromConfigFile(configFile)

		if !backend.SupportsQuickCheck() {
			reportCLIErrorAndExit(fmt.Errorf("backend '%v' does not support quick check", backend.ConfigType()))
			return
		}

		if err := backend.QuickCheck(configFile); err != nil {
			reportCLIErrorAndExit(err)
			return
		}

	},
}

func init() {
	rootCmd.AddCommand(quickCheckCmd)

	quickCheckCmd.Args = func(cmd *cobra.Command, args []string) error {

		// if len(args) != 1 {
		// 	return fmt.Errorf("one argument required: (config file path)")
		// }

		return nil
	}
}
