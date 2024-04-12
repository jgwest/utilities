package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		pathToConfigFile := args[0]
		outputPath := args[1]

		backend := retrieveBackendFromConfigFile(pathToConfigFile)

		if !backend.SupportsGenerateBackup() {
			reportCLIErrorAndExit(fmt.Errorf("backend '%v' does not support generating backup files", backend.ConfigType()))
			return
		}

		if err := backend.GenerateBackup(pathToConfigFile, outputPath); err != nil {
			reportCLIErrorAndExit(err)
			return
		}

	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Args = func(cmd *cobra.Command, args []string) error {

		if len(args) != 2 {
			return fmt.Errorf("arguments required: (path to yaml file) (output path)")
		}

		return nil
	}

}
