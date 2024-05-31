package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jgwest/backup-cli/backends"
	"github.com/jgwest/backup-cli/model"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "newApp",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) {
	// 	thing()
	// },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.newApp.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".newApp" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".newApp")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func reportCLIErrorAndExit(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func retrieveBackendFromConfigFile(pathToConfigFile string) model.Backend {
	model, err := model.ReadConfigFile(pathToConfigFile)
	if err != nil {
		reportCLIErrorAndExit(err)
		return nil
	}

	backend, err := findBackendForConfigFile(model)
	if err != nil {
		reportCLIErrorAndExit(fmt.Errorf("unable to locate backend implementation for '%s': %w", pathToConfigFile, err))
		return nil
	}

	return backend
}

func findConfigFile() (string, error) {

	fileinfoList, err := os.ReadDir(".")
	if err != nil {
		return "", err
	}

	matches := []string{}

	for _, info := range fileinfoList {

		if info.IsDir() {
			continue
		}

		if !strings.HasSuffix(strings.ToLower(info.Name()), ".yaml") {
			continue
		}

		matches = append(matches, info.Name())
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple YAML files in folder")
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no YAML files in folder")
	}

	return matches[0], nil

}

// If the command only takes a single param, which is the path to yaml, then you can call this to retrieve it
func getOptionalConfigFilePath(args []string) string {
	var configFile string

	if len(args) == 1 && strings.HasSuffix(args[0], ".yaml") {
		configFile = args[0]
	} else if len(args) == 0 {
		var err error
		configFile, err = findConfigFile()
		if err != nil {
			reportCLIErrorAndExit(err)
			return ""
		}
	} else {
		reportCLIErrorAndExit(fmt.Errorf("unexpected args"))
		return ""
	}

	return configFile

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
