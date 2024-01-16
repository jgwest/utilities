package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jgwest/backup-cli/model"
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

		model, err := model.ReadConfigFile(configFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}

		backend, err := findBackendForConfigFile(model)
		if err != nil {
			fmt.Printf("unable to locate backend implementation for '%s'\n", configFile)
			os.Exit(1)
			return
		}

		if err := backend.QuickCheck(configFile); err != nil {
			fmt.Println(err)
			os.Exit(1)
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
