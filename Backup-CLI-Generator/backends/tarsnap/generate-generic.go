package tarsnap

import (
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds"
)

func (r TarsnapBackend) SupportsGenerateGeneric() bool {
	return true
}

func (r TarsnapBackend) GenerateGeneric(path string, outputPath string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	result, err := generateGenericScriptFromConfigFile(path, config, false)
	if err != nil {
		return err
	}

	// If the output path already exists, don't overwrite it
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("output path already exists: %s", outputPath)
	}

	if err := os.WriteFile(outputPath, []byte(result), 0700); err != nil {
		return err
	}

	fmt.Println("output: " + result)

	return nil
}

func generateGenericScriptFromConfigFile(configFilePath string, config model.ConfigFile, dryRun bool) (string, error) {

	nodes := util.NewTextNodes()

	cmds.AddGenericPrefixNode(nodes)

	if err := generateGenericInvocationNode(config, nodes); err != nil {
		return "", err
	}

	return nodes.ToString()

}

func generateGenericInvocationNode(config model.ConfigFile, textNodes *util.TextNodes) error {

	tarsnapCredentials, err := config.GetTarsnapCredential()
	if err != nil {
		return err
	}

	if _, err := os.Stat(tarsnapCredentials.ConfigFilePath); os.IsNotExist(err) {
		return fmt.Errorf("tarsnap config path does not exist: '%s'", tarsnapCredentials.ConfigFilePath)
	}

	invocation := textNodes.NewTextNode()

	invocation.Out()
	invocation.Header("Invocation")

	additionalParams := ""
	if textNodes.IsWindows() {
		additionalParams = "%*"

	} else {
		additionalParams = "$*"
	}

	cliInvocation := fmt.Sprintf(
		"tarsnap --humanize-numbers --configfile \"%s\" %s",
		tarsnapCredentials.ConfigFilePath,
		additionalParams)

	invocation.Out()

	if textNodes.IsWindows() {
		invocation.Out(cliInvocation)
	} else {
		invocation.Out("bash -c \"" + cliInvocation + "\"")
	}

	return nil
}
