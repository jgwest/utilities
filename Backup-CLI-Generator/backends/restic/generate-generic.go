package restic

import (
	"errors"
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func (r ResticBackend) SupportsGenerateGeneric() bool {
	return true
}

func (r ResticBackend) GenerateGeneric(path string, outputPath string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	result, err := generateGenericProcessConfig(path, config, false)
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

func generateGenericProcessConfig(configFilePath string, config model.ConfigFile, dryRun bool) (string, error) {

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

	if err := resticGenerateGenericInvocation2(config, nodes); err != nil {
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
		if err := SharedGenerateResticCredentials(config, credentials); err != nil {
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

func SharedGenerateResticCredentials(config model.ConfigFile, node *util.TextNode) error {

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return err
	}
	node.Out()
	node.Header("Credentials ")

	if resticCredential.S3 != nil {
		node.SetEnv("AWS_ACCESS_KEY_ID", resticCredential.S3.AccessKeyID)
		node.SetEnv("AWS_SECRET_ACCESS_KEY", resticCredential.S3.SecretAccessKey)
	}

	if len(resticCredential.Password) > 0 && len(resticCredential.PasswordFile) > 0 {
		return errors.New("both password and password file are specified")
	}

	if len(resticCredential.Password) > 0 {
		node.SetEnv("RESTIC_PASSWORD", resticCredential.Password)

	} else if len(resticCredential.PasswordFile) > 0 {
		node.SetEnv("RESTIC_PASSWORD_FILE", resticCredential.PasswordFile)

	} else {
		return errors.New("no restic password found")
	}

	return nil

}
