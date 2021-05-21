package cmd

import (
	"fmt"

	"github.com/jgwest/backup-cli/check"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Output a diff between the expected script, and the actual script.",
	Long:  "Output a diff between the expected script, and the actual script.",
	Run: func(cmd *cobra.Command, args []string) {
		check.RunCheck(args[0], args[1])
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
