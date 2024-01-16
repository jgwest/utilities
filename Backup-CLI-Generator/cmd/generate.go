package cmd

import (
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/model"
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

		model, err := model.ReadConfigFile(pathToConfigFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
			return
		}

		backend, err := findBackendForConfigFile(model)
		if err != nil {
			fmt.Printf("unable to locate backend implementation for '%s'\n", pathToConfigFile)
			os.Exit(1)
			return
		}

		if err := backend.GenerateBackup(pathToConfigFile, outputPath); err != nil {
			fmt.Println(err)
			os.Exit(1)
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
