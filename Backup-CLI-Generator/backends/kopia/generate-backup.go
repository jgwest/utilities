package kopia

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds"
	"github.com/jgwest/backup-cli/util/cmds/generate"
)

func (KopiaBackend) SupportsGenerateBackup() bool {
	return true
}

func (KopiaBackend) GenerateBackup(path string, outputPath string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	result, err := generateBackupScriptFromConfigFile(path, config)
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

func generateBackupScriptFromConfigFile(configFilePath string, config model.ConfigFile) (string, error) {

	if err := generate.CheckMonitorFoldersForMissingChildren(configFilePath, config); err != nil {
		return "", err
	}

	nodes := util.NewTextNodes()

	cmds.AddGenericPrefixNode(nodes)

	// Add BACKUP_DATA_TIME env var, if required
	if config.Metadata != nil {

		if config.Metadata.Name == "" {
			return "", fmt.Errorf("if metadata is specified, then name must be specified")
		}

		if config.Metadata.AppendDateTime {
			backupDateTime := nodes.NewTextNode()

			if nodes.IsWindows() {
				backupDateTime.Out("set BACKUP_DATE_TIME=%DATE%-%TIME:~1%")
			} else {
				backupDateTime.Out("BACKUP_DATE_TIME=`date +%F_%H:%M:%S`")
			}

			backupDateTime.AddExports("BACKUP_DATE_TIME")
		}
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

			// TODO: Kopia: This needs to be something different on Windows, probably without the slash
			excludesNode.SetEnv("EXCLUDES", substring+"--add-ignore \\\""+expandedValue+"\\\"")

			if nodes.IsWindows() {
				return "", fmt.Errorf("this needs to be something different on Windows, probably without the slash")
			}

		}
	}

	// key: path to be backed up
	// value: list of excludes for that path
	kopiaPolicyExcludes := map[string][]string{}

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
		processedFolders, err := generate.PopulateProcessedFolders(model.Kopia, config.Folders, config.Substitutions, kopiaPolicyExcludes)
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

	// Uses TODO, BACKUP_DATE_TIME, EXCLUDES, from above
	invocationNode, err := generateBackupInvocationNode(kopiaPolicyExcludes, config, nodes)
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

func generateBackupInvocationNode(kopiaPolicyExcludes map[string][]string, config model.ConfigFile, textNodes *util.TextNodes) (*util.TextNode, error) {

	textNode := textNodes.NewTextNode()

	kopiaCredentials, err := getAndValidateKopiaCredentials(config)
	if err != nil {
		return nil, err
	}

	// Set credentials env vars
	{

		textNode.Out()
		textNode.Header("Credentials")
		textNode.SetEnv("AWS_ACCESS_KEY_ID", kopiaCredentials.S3.AccessKeyID)
		textNode.SetEnv("AWS_SECRET_ACCESS_KEY", kopiaCredentials.S3.SecretAccessKey)

		if len(kopiaCredentials.Password) > 0 {
			textNode.SetEnv("KOPIA_PASSWORD", kopiaCredentials.Password)
		}
	}

	textNode.Out()
	textNode.Header("Connect repository")

	cliInvocation := fmt.Sprintf("kopia repository connect s3 --bucket=\"%s\" --access-key=\"%s\" --secret-access-key=\"%s\" --password=\"%s\" --endpoint=\"%s\"",
		kopiaCredentials.KopiaS3.Bucket,
		textNode.Env("AWS_ACCESS_KEY_ID"),
		textNode.Env("AWS_SECRET_ACCESS_KEY"),
		textNode.Env("KOPIA_PASSWORD"),
		kopiaCredentials.KopiaS3.Endpoint)

	textNode.Out(cliInvocation)

	if len(config.GlobalExcludes) > 0 {
		cliInvocation = fmt.Sprintf("kopia policy set --global %s", textNode.Env("EXCLUDES"))
		textNode.Out(cliInvocation)
	}

	// For each local path, call set policy with the ignores for that path
	if len(kopiaPolicyExcludes) > 0 {
		textNode.Out()
		textNode.Header("Add policy excludes")

		for backupPath, excludes := range kopiaPolicyExcludes {

			if len(excludes) == 0 {
				continue
			}

			excludesStr := ""
			for _, exclude := range excludes {
				excludesStr += "--add-ignore \"" + exclude + "\" "
			}
			excludesStr = strings.TrimSpace(excludesStr)

			cliInvocation = fmt.Sprintf("kopia policy set %s \"%s\"", excludesStr, backupPath)
			textNode.Out(cliInvocation)
		}
	}

	textNode.Out()
	textNode.Header("Create snapshot")

	descriptionSubstring := ""
	if config.Metadata != nil && config.Metadata.Name != "" {
		description := config.Metadata.Name

		if config.Metadata.AppendDateTime {
			description += textNode.Env("BACKUP_DATE_TIME")
		}

		descriptionSubstring = fmt.Sprintf("--description=\"%s\" ", description)
	}

	cliInvocation = fmt.Sprintf("kopia snapshot create %s%s",
		descriptionSubstring,
		textNode.Env("TODO"))

	if textNodes.IsWindows() {
		textNode.Out(cliInvocation)
	} else {
		textNode.Out("bash -c \"" + cliInvocation + "\"")
	}

	return textNode, nil
}
