package tarsnap

import (
	"fmt"
	"os"
	"runtime"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func (r TarsnapBackend) SupportsGenerateGeneric() bool {
	return true
}

func (r TarsnapBackend) GenerateGeneric(path string, outputPath string) error {

	config, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	if configType, err := config.GetConfigType(); err != nil {
		return err
	} else if configType != model.Tarsnap {

		return fmt.Errorf("configuration file does not support tarnsap")
	}

	resultBuffer, err := ProcessConfig(path, config, false)
	if err != nil {
		return err
	}

	result := resultBuffer.ToString()

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

func ProcessConfig(configFilePath string, config model.ConfigFile, dryRun bool) (*util.OutputBuffer, error) {

	buffer := util.OutputBuffer{
		IsWindows: runtime.GOOS == "windows",
	}

	if buffer.IsWindows {
		buffer.Lines = []string{"@echo off", "setlocal"}
		// https://stackoverflow.com/questions/17063947/get-current-batchfile-directory
		buffer.Out("set SCRIPTPATH=\"%~f0\"")
	} else {
		buffer.Lines = []string{"#!/bin/bash", "", "set -eu"}
		// https://stackoverflow.com/questions/4774054/reliable-way-for-a-bash-script-to-get-the-full-path-to-itself
		buffer.Out("SCRIPTPATH=`realpath -s $0`")
	}

	if err := tarsnapGenerateGenericInvocation(config, &buffer); err != nil {
		return nil, err
	}

	return &buffer, nil
}

func tarsnapGenerateGenericInvocation(config model.ConfigFile, buffer *util.OutputBuffer) error {

	tarsnapCredentials, err := config.GetTarsnapCredential()
	if err != nil {
		return err
	}

	if _, err := os.Stat(tarsnapCredentials.ConfigFilePath); os.IsNotExist(err) {
		return fmt.Errorf("tarsnap config path does not exist: '%s'", tarsnapCredentials.ConfigFilePath)
	}

	additionalParams := ""
	if buffer.IsWindows {
		additionalParams = "%*"
	} else {
		additionalParams = "$*"
	}

	cliInvocation := fmt.Sprintf(
		"tarsnap --humanize-numbers --configfile \"%s\" %s",
		tarsnapCredentials.ConfigFilePath,
		additionalParams)

	buffer.Out()

	if buffer.IsWindows {
		buffer.Out(cliInvocation)
	} else {
		buffer.Out("bash -c \"" + cliInvocation + "\"")
	}

	return nil
}
