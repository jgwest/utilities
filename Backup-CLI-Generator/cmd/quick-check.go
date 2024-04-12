package cmd

import (
	"fmt"
	"strings"

	"github.com/jgwest/backup-cli/util"
	"github.com/spf13/cobra"
)

// quickCheckCmd represents the check command
var quickCheckCmd = &cobra.Command{
	Use:   "quick-check",
	Short: "...",
	Long:  "...",
	Run: func(cmd *cobra.Command, args []string) {

		var configFile string

		if len(args) == 1 && strings.HasSuffix(args[0], ".yaml") {
			configFile = args[0]
		} else if len(args) == 0 {
			var err error
			configFile, err = util.FindConfigFile()
			if err != nil {
				reportCLIErrorAndExit(err)
				return
			}
		} else {
			reportCLIErrorAndExit(fmt.Errorf("unexpected args"))
			return
		}

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
