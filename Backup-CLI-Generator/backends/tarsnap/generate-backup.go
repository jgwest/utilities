package tarsnap

import (
	"errors"
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/generate"
	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func (r TarsnapBackend) SupportsGenerateBackup() bool {
	return true
}

func (r TarsnapBackend) GenerateBackup(path string, outputPath string) error {

	config, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	if configType != model.Tarsnap {
		return fmt.Errorf("configuration file does not support tarsnap backend")
	}

	// TODO: Re-enable dryrun on tarsnap
	result, err := processGenerateBackupConfig(path, config, false)
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

	fmt.Println(result)

	return nil
}

func processGenerateBackupConfig(configFilePath string, config model.ConfigFile, dryRun bool) (string, error) {

	if err := generate.CheckMonitorFolders(configFilePath, config); err != nil {
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
	prefixNode.AddExports("SCRIPTPATH")

	if config.Metadata != nil {

		backupDateTime := nodes.NewTextNode()

		if config.Metadata.Name == "" {
			return "", fmt.Errorf("if metadata is specified, then name must be specified")
		}

		if config.Metadata.AppendDateTime {
			if nodes.IsWindows() {
				backupDateTime.Out("set BACKUP_DATE_TIME=%DATE%-%TIME:~1%")
			} else {
				backupDateTime.Out("BACKUP_DATE_TIME=`date +%F_%H:%M:%S`")
			}
		}
		backupDateTime.AddExports("BACKUP_DATE_TIME")
	}

	excludesNode := nodes.NewTextNode()

	// Populate EXCLUDES var, by processing Global Excludes
	if len(config.GlobalExcludes) > 0 {

		excludesNode.Out()
		excludesNode.Header("Excludes")
		for index, exclude := range config.GlobalExcludes {

			substring := ""

			if index > 0 {
				substring = excludesNode.Env("EXCLUDES") + " "
			}

			expandedValue, err := util.Expand(exclude, config.Substitutions)
			if err != nil {
				return "", err
			}

			if nodes.IsWindows() {
				excludesNode.SetEnv("EXCLUDES", substring+"--exclude \""+expandedValue+"\"")
			} else {
				excludesNode.SetEnv("EXCLUDES", substring+"--exclude \\\""+expandedValue+"\\\"")
			}
		}
	}

	// Robocopy only: Populate EXCLUDES
	if config.RobocopySettings != nil {
		return "", errors.New("robocopy settings are not supported in tarsnap backend")
	}

	// Process folders
	// - Populate TODO env var, for everything except robocopy
	// - For robocopy, populate robocopyFolders
	{
		foldersNode := nodes.NewTextNode()

		if len(config.Folders) == 0 {
			return "", errors.New("at least one folder is required")
		}

		foldersNode.Out("")
		foldersNode.Header("Folders")

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		// - This function also updates kopiaPolicyExcludes, if applicable.
		processedFolders, err := generate.PopulateProcessedFolders(model.Tarsnap, config.Folders, config.Substitutions, map[string][]string{})
		if err != nil {
			return "", fmt.Errorf("unable to populateProcessedFolder: %v", err)
		}

		for index, processedFolder := range processedFolders {
			substring := ""

			if index > 0 {
				substring = foldersNode.Env("TODO") + " "
			}

			folderPath, ok := (processedFolder[0]).(string)
			if !ok {
				return "", fmt.Errorf("invalid non-robocopy folderPath")
			}

			// TODO: This needs to be something different on Windows, probably without the slash

			// The unsubstituted path is used here

			if nodes.IsWindows() {
				foldersNode.SetEnv("TODO", fmt.Sprintf("%s\"%s\"", substring, folderPath))
			} else {
				foldersNode.SetEnv("TODO", fmt.Sprintf("%s\\\"%s\\\"", substring, folderPath))
			}

		}

	} // end 'process folders' section

	// Uses TODO, EXCLUDES, BACKUP_DATE_TIME, from above
	invocationNode, err := tarsnapGenerateInvocation2(config, dryRun, nodes)
	if err != nil {
		return "", err
	}
	suffixNode := nodes.NewTextNode()
	suffixNode.Out()
	suffixNode.Header("Verify the YAML file still produces this script")
	suffixNode.Out("backup-cli check \"" + configFilePath + "\" " + suffixNode.Env("SCRIPTPATH"))
	suffixNode.AddDependency(invocationNode)

	return nodes.ToString()
}

func tarsnapGenerateInvocation2(config model.ConfigFile, dryRun bool, textNodes *util.TextNodes) (*util.TextNode, error) {

	textNode := textNodes.NewTextNode()

	tarsnapCredentials, err := config.GetTarsnapCredential()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(tarsnapCredentials.ConfigFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("tarsnap config path does not exist: '%s'", tarsnapCredentials.ConfigFilePath)
	}

	if config.Metadata == nil || len(config.Metadata.Name) == 0 {
		return nil, fmt.Errorf("tarsnap requires a metadata name")
	}

	backupName := config.Metadata.Name
	if config.Metadata.AppendDateTime {
		backupName += textNode.Env("BACKUP_DATE_TIME")
	}

	dryRunSubstring := ""
	if dryRun {
		dryRunSubstring = "--dry-run "
	}

	excludesSubstring := ""
	if len(config.GlobalExcludes) > 0 {
		excludesSubstring = textNode.Env("EXCLUDES") + " "
	}

	cliInvocation := fmt.Sprintf(
		"tarsnap --humanize-numbers --configfile \"%s\" -c %s%s -f \"%s\" %s",
		tarsnapCredentials.ConfigFilePath,
		dryRunSubstring,
		excludesSubstring,
		backupName,
		textNode.Env("TODO"))

	textNode.Out()

	if textNodes.IsWindows() {
		textNode.Out(cliInvocation)
	} else {
		textNode.Out("bash -c \"" + cliInvocation + "\"")
	}

	return textNode, nil
}
