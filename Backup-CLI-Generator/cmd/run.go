package cmd

import (
	"fmt"
	"strings"

	"github.com/jgwest/backup-cli/run"
	"github.com/jgwest/backup-cli/util"
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		configFile := ""
		var params []string

		if len(args) >= 1 && strings.HasSuffix(args[0], ".yaml") {
			configFile = args[0]
			params = args[1:]
		} else {
			var err error
			configFile, err = util.FindConfigFile()
			if err != nil {
				fmt.Println(err)
				return
			}

			params = args[0:]
		}

		if err := run.Run(configFile, params); err != nil {
			fmt.Println(err)
			return
		}

	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// generateCmd.Args = func(cmd *cobra.Command, args []string) error {

	// 	if len(args) != 2 {
	// 		return fmt.Errorf("arguments required: (path to yaml file) (output path)")
	// 	}

	// 	return nil
	// }

}
