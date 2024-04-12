package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var genericCmd = &cobra.Command{
	Use:   "generic",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 2 {
			fmt.Println("args: (path to config file) (output path)")
			return
		}

		pathToConfigFile := args[0]
		outputPath := args[1]

		backend := retrieveBackendFromConfigFile(pathToConfigFile)

		if !backend.SupportsGenerateGeneric() {
			reportCLIErrorAndExit(fmt.Errorf("backend '%v' does not support generic generation", backend.ConfigType()))
			return
		}

		if err := backend.GenerateGeneric(pathToConfigFile, outputPath); err != nil {
			reportCLIErrorAndExit(err)
			return
		}

	},
}

func init() {
	generateCmd.AddCommand(genericCmd)

}
