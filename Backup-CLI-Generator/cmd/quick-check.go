package cmd

import (
	"fmt"
	"strings"

	"github.com/jgwest/backup-cli/quickcheck"
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
				fmt.Println(err)
				return
			}
		} else {
			fmt.Println("unexpected args")
			return
		}

		err := quickcheck.RunQuickCheck(configFile)
		if err != nil {
			fmt.Println(err)
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
