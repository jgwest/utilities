package restic

import (
	"errors"
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds"
)

func (ResticBackend) SupportsGenerateGeneric() bool {
	return true
}

func (ResticBackend) GenerateGeneric(path string, outputPath string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	result, err := generateGenericScriptFromConfigFile(path, config)
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

func generateGenericScriptFromConfigFile(configFilePath string, config model.ConfigFile) (string, error) {

	nodes := util.NewTextNodes()

	cmds.AddGenericPrefixNode(nodes)

	if err := generateGenericInvocationNode(config, nodes); err != nil {
		return "", err
	}

	return nodes.ToString()

}

func generateGenericInvocationNode(config model.ConfigFile, textNodes *util.TextNodes) error {

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return err
	}

	// Build credentials nodes
	credentials := textNodes.NewTextNode()
	{
		if err := sharedGenerateResticCredentials(config, credentials); err != nil {
			return err
		}
	}

	invocation := textNodes.NewTextNode()
	invocation.AddDependency(credentials)

	invocation.Out()
	invocation.Header("Invocation")

	url := ""
	if resticCredential.S3 != nil {
		url = "s3:" + resticCredential.S3.URL
	} else if resticCredential.RESTEndpoint != "" {
		url = "rest:" + resticCredential.RESTEndpoint
	} else {
		return errors.New("unable to locate connection credentials")
	}

	cacertSubstring := ""
	if resticCredential.CACert != "" {
		expandedPath, err := util.Expand(resticCredential.CACert, config.Substitutions)
		if err != nil {
			return err
		}
		cacertSubstring = "--cacert \"" + expandedPath + "\" "
	}

	additionalParams := ""
	if textNodes.IsWindows() {
		additionalParams = "%*"

	} else {
		additionalParams = "$*"
	}

	cliInvocation := fmt.Sprintf("restic -r %s --verbose %s %s",
		url,
		cacertSubstring,
		additionalParams)

	invocation.Out()

	if textNodes.IsWindows() {
		invocation.Out(cliInvocation)
	} else {
		invocation.Out("bash -c \"" + cliInvocation + "\"")
	}

	return nil
}
