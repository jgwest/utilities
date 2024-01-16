package restic

import (
	"errors"
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/generic"
	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func (r ResticBackend) SupportsGenerateGeneric() bool {
	return true
}

func (r ResticBackend) GenerateGeneric(path string, outputPath string) error {

	model, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	result, err := GenerateGenericProcessConfig(path, model, false)
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

func GenerateGenericProcessConfig(configFilePath string, config model.ConfigFile, dryRun bool) (string, error) {

	configType, err := config.GetConfigType()
	if err != nil {
		return "", err
	}

	nodes := util.NewTextNodes()

	prefixNode := nodes.NewPrefixTextNode()

	if nodes.IsWindows() {
		// https://stackoverflow.com/questions/17063947/get-current-batchfile-directory
		prefixNode.Out("@echo off", "setlocal")
		prefixNode.Out("set SCRIPTPATH=\"%~f0\"")
	} else {
		prefixNode.Out("#!/bin/bash", "", "set -eu")
		// https://stackoverflow.com/questions/4774054/reliable-way-for-a-bash-script-to-get-the-full-path-to-itself
		prefixNode.Out("SCRIPTPATH=`realpath -s $0`")
	}

	if configType == model.Restic {
		err = resticGenerateGenericInvocation2(config, nodes)
	} else if configType == model.Kopia {
		// 	err = kopiaGenerateGenericInvocation(config, &buffer)
		// } else if configType == model.Tarsnap {
		// 	err = tarsnapGenerateGenericInvocation(config, &buffer)
		// } else {
		return "", fmt.Errorf("unsupported configType: %v", configType)
	}

	if err != nil {
		return "", err
	}

	return nodes.ToString()

}

func resticGenerateGenericInvocation2(config model.ConfigFile, textNodes *util.TextNodes) error {

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return err
	}

	// Build credentials nodes
	credentials := textNodes.NewTextNode()
	{
		if err := generic.SharedGenerateResticCredentials(config, credentials); err != nil {
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
