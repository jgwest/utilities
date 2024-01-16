package restic

import (
	"errors"
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/generate"
	"github.com/jgwest/backup-cli/generic"
	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func (r ResticBackend) SupportsGenerateBackup() bool {
	return true
}

func (r ResticBackend) GenerateBackup(path string, outputPath string) error {

	model, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	result, err := ProcessConfigGenerateBackup(path, model, false)
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

func ProcessConfigGenerateBackup(configFilePath string, config model.ConfigFile, dryRun bool) (string, error) {

	configType, err := config.GetConfigType()
	if err != nil {
		return "", err
	}

	if configType != model.Restic {
		return "", fmt.Errorf("unsupported config type: %v", configType)
	}

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

	if config.RobocopySettings != nil {
		return "", fmt.Errorf("robocopy setting is defined, but is not supported by this backend")
	}

	// Process folders
	// - Populate TODO env var, for everything except robocopy
	{
		foldersNode := nodes.NewTextNode()

		if len(config.Folders) == 0 {
			return "", errors.New("at least one folder is required")
		}

		foldersNode.Out("")
		foldersNode.Header("Folders")

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		processedFolders, err := generate.PopulateProcessedFolders(configType, config.Folders, config.Substitutions, map[string][]string{})
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

	var invocationNode *util.TextNode

	// Uses the 'TODO' env var, generated above, to know what to backup.
	invocationNode, err = resticGenerateBackupInvocation2(config, nodes)
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

func resticGenerateBackupInvocation2(config model.ConfigFile, textNodes *util.TextNodes) (*util.TextNode, error) {

	{
		credentialsNode := textNodes.NewTextNode()

		if err := generic.SharedGenerateResticCredentials(config, credentialsNode); err != nil {
			return nil, err
		}
	}

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return nil, err
	}

	invocationTextNode := textNodes.NewTextNode()

	tagSubstring := ""
	if config.Metadata != nil {
		if len(config.Metadata.Name) == 0 {
			return nil, errors.New("metadata exists, but name is nil")
		}

		quote := "'"
		if textNodes.IsWindows() {
			quote = "\""
		}

		tagSubstring = fmt.Sprintf("--tag %s%s", quote, config.Metadata.Name)
		if config.Metadata.AppendDateTime {
			tagSubstring += invocationTextNode.Env("BACKUP_DATE_TIME")
		}

		tagSubstring += quote + " "
	}

	url := ""
	if resticCredential.S3 != nil {
		url = "s3:" + resticCredential.S3.URL
	} else if resticCredential.RESTEndpoint != "" {
		url = "rest:" + resticCredential.RESTEndpoint
	} else {
		return nil, errors.New("unable to locate connection credentials")
	}

	cacertSubstring := ""
	if resticCredential.CACert != "" {
		expandedPath, err := util.Expand(resticCredential.CACert, config.Substitutions)
		if err != nil {
			return nil, err
		}
		cacertSubstring = "--cacert \"" + expandedPath + "\" "
	}

	excludesSubstring := ""
	if len(config.GlobalExcludes) > 0 {
		excludesSubstring = invocationTextNode.Env("EXCLUDES") + " "
	}

	cliInvocation := fmt.Sprintf("restic -r %s --verbose %s%s%s backup %s",
		url,
		tagSubstring,
		cacertSubstring,
		excludesSubstring,
		invocationTextNode.Env("TODO"))

	invocationTextNode.Out()

	if textNodes.IsWindows() {
		invocationTextNode.Out(cliInvocation)
	} else {
		invocationTextNode.Out("bash -c \"" + cliInvocation + "\"")
	}

	return invocationTextNode, nil
}
