package cmd

import (
	"fmt"
	"strings"

	"github.com/jgwest/backup-cli/backends"
	"github.com/jgwest/backup-cli/model"
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
				reportCLIErrorAndExit(err)
				return
			}

			params = args[0:]
		}

		backend := retrieveBackendFromConfigFile(configFile)

		if !backend.SupportsRun() {
			reportCLIErrorAndExit(fmt.Errorf("backend '%v' does not support run", backend.ConfigType()))
			return
		}

		if err := backend.Run(configFile, params); err != nil {
			reportCLIErrorAndExit(err)
			return

		}

	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func findBackendForConfigFile(config model.ConfigFile) (model.Backend, error) {
	availableBackends := backends.AvailableBackends()

	configType, err := config.GetConfigType()
	if err != nil {
		return nil, fmt.Errorf("unable to extract config type: %v", err)
	}

	for i := range availableBackends {
		backend := availableBackends[i]

		if backend.ConfigType() == configType {
			return backend, nil
		}

	}

	return nil, fmt.Errorf("supported backend for '%v' not found", configType)

}
