package cmd

import (
	"fmt"

	"github.com/jgwest/backup-cli/quickcheck"
	"github.com/spf13/cobra"
)

// quickCheckCmd represents the check command
var quickCheckCmd = &cobra.Command{
	Use:   "quick-check",
	Short: "...",
	Long:  "...",
	Run: func(cmd *cobra.Command, args []string) {
		err := quickcheck.RunQuickCheck(args[0])
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(quickCheckCmd)

	quickCheckCmd.Args = func(cmd *cobra.Command, args []string) error {

		if len(args) != 1 {
			return fmt.Errorf("one argument required: (config file path)")
		}

		return nil
	}
}
