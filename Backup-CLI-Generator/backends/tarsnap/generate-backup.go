package tarsnap

import (
	"errors"
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds"
	"github.com/jgwest/backup-cli/util/cmds/generate"
)

func (TarsnapBackend) SupportsGenerateBackup() bool {
	return true
}

func (TarsnapBackend) GenerateBackup(path string, outputPath string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	// TODO: Re-enable dryrun on tarsnap
	result, err := generateBackupScriptFromConfigFile(path, config, false)
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

func generateBackupScriptFromConfigFile(configFilePath string, config model.ConfigFile, dryRun bool) (string, error) {

	if err := generate.CheckMonitorFoldersForMissingChildren(configFilePath, config); err != nil {
		return "", err
	}

	nodes := util.NewTextNodes()

	cmds.AddGenericPrefixNode(nodes)

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

	// Process folders
	// - Populate TODO env var
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

			folderPath := processedFolder.SrcFolderPath

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
	invocationNode, err := generateBackupInvocationNode(config, dryRun, nodes)
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

func generateBackupInvocationNode(config model.ConfigFile, dryRun bool, textNodes *util.TextNodes) (*util.TextNode, error) {

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
