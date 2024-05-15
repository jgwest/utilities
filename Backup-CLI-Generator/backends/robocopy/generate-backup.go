package robocopy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds"
	"github.com/jgwest/backup-cli/util/cmds/generate"
)

func (r RobocopyBackend) SupportsGenerateBackup() bool {
	return true
}

func (r RobocopyBackend) GenerateBackup(path string, outputPath string) error {

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

		return "", errors.New("robocopy does not support global excludes")

	}

	// Robocopy only: Populate EXCLUDES
	if config.RobocopySettings != nil {

		if !nodes.IsWindows() {
			return "", errors.New("robocopy settings not supported for non-windows")
		}

		excludesNode.Out()
		excludesNode.Header("Excludes")

		excludesCount := 0

		for _, excludeFile := range config.RobocopySettings.ExcludeFiles {

			substring := ""

			if excludesCount > 0 {
				substring = excludesNode.Env("EXCLUDES") + " "
			}

			expandedValue, err := util.Expand(excludeFile, config.Substitutions)
			if err != nil {
				return "", err
			}

			excludesNode.SetEnv("EXCLUDES", substring+"/XF \""+expandedValue+"\"")

			excludesCount++
		}

		for _, excludeDir := range config.RobocopySettings.ExcludeFolders {

			substring := ""

			if excludesCount > 0 {
				substring = excludesNode.Env("EXCLUDES") + " "
			}

			expandedValue, err := util.Expand(excludeDir, config.Substitutions)
			if err != nil {
				return "", err
			}

			if strings.Contains(expandedValue, "*") {
				return "", fmt.Errorf("wildcards may not be supported in directories with robocopy: %s", expandedValue)
			}

			excludesNode.SetEnv("EXCLUDES", substring+"/XD \""+expandedValue+"\"")

			excludesCount++
		}

	}

	// robocopyFolders contains a slice of:
	// - source folder path
	// - destination folder (with basename of source folder appended)
	// Example:
	// - [C:\Users] -> [B:\backup\C-Users]
	// - [D:\Users] -> [B:\backup\D-Users]
	// - [C:\To-Backup] -> [B:\backup\To-Backup]
	var robocopyFolders [][]string

	// Process folders
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
		processedFolders, err := generate.PopulateProcessedFolders(model.Robocopy, config.Folders, config.Substitutions, map[string][]string{})
		if err != nil {
			return "", fmt.Errorf("unable to populateProcessedFolder: %v", err)
		}

		// Ensure that none of the folders share a basename
		if err := robocopyValidateBasenames(processedFolders); err != nil {
			return "", err
		}

		if robocopyCredentials, err := config.GetRobocopyCredential(); err == nil {

			robocopyFolders, err = robocopyGenerateTargetPaths(processedFolders, robocopyCredentials)
			if err != nil {
				return "", err
			}

		} else {
			return "", err
		}

	} // end 'process folders' section

	invocationNode, err := generateBackupInvocationNode(config, robocopyFolders, nodes)
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

func getAndValidateRobocopyCredentials(config model.ConfigFile) (*model.RobocopyCredentials, error) {
	robocopyCredentials, err := config.GetRobocopyCredential()
	if err != nil {
		return nil, err
	}

	if robocopyCredentials.DestinationFolder == "" {
		return nil, errors.New("missing destination folder")
	}

	if robocopyCredentials.Switches == "" {
		return nil, errors.New("missing switches")
	}

	if config.Metadata != nil && (config.Metadata.Name != "" || config.Metadata.AppendDateTime) {
		return nil, fmt.Errorf("metadata features are not supported with robocopy")
	}

	if _, err := os.Stat(robocopyCredentials.DestinationFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("robocopy destination folder does not exist: '%s'", robocopyCredentials.DestinationFolder)
	}

	return &robocopyCredentials, nil

}

func generateBackupInvocationNode(config model.ConfigFile, robocopyFolders [][]string, textNodes *util.TextNodes) (*util.TextNode, error) {

	robocopyCredentials, err := getAndValidateRobocopyCredentials(config)
	if err != nil {
		return nil, err
	}

	textNode := textNodes.NewTextNode()

	envSwitch := ""

	if config.RobocopySettings != nil && (len(config.RobocopySettings.ExcludeFiles) > 0 || len(config.RobocopySettings.ExcludeFolders) > 0) {
		envSwitch += " " + textNode.Env("EXCLUDES")
	}

	textNode.SetEnv("SWITCHES", robocopyCredentials.Switches+envSwitch)

	for _, folderTuple := range robocopyFolders {
		srcFolder := util.FixWindowsPathSuffix("\"" + folderTuple[0] + "\"")
		destFolder := util.FixWindowsPathSuffix("\"" + folderTuple[1] + "\"")
		textNode.Out(fmt.Sprintf("robocopy %s %s %s", srcFolder, destFolder, textNode.Env("SWITCHES")))
	}

	return textNode, nil
}

// robocopyGenerateTargetPaths returns a slice of:
// - source folder path
// - destination folder (with basename of source folder appended)
// Example:
// - [C:\Users] -> [B:\backup\C-Users]
// - [D:\Users] -> [B:\backup\D-Users]
// - [C:\To-Backup] -> [B:\backup\To-Backup]
func robocopyGenerateTargetPaths(processedFolders [][]interface{}, robocopyCredentials model.RobocopyCredentials) ([][]string, error) {
	res := [][]string{}

	targetFolder := robocopyCredentials.DestinationFolder

	for _, robocopyFolder := range processedFolders {

		robocopySrcFolderPath, ok := (robocopyFolder[0]).(string)
		if !ok {
			return nil, fmt.Errorf("invalid robocopyFolderPath")
		}

		folderEntry, ok := (robocopyFolder[1]).(model.Folder)
		if !ok {
			return nil, fmt.Errorf("invalid robocopyFolder")
		}

		// Use the name of the src folder as the dest folder name, unless
		// a replacement is specified in the folder entry.
		destFolderName := filepath.Base(robocopySrcFolderPath)
		if folderEntry.Robocopy != nil && folderEntry.Robocopy.DestFolderName != "" {
			destFolderName = folderEntry.Robocopy.DestFolderName
		}

		// tuple:
		// - source folder path
		// - destination folder with basename of source folder appended
		tuple := []string{robocopySrcFolderPath, filepath.Join(targetFolder, destFolderName)}

		res = append(res, tuple)

	}

	return res, nil

}

// robocopyValidateBasenames ensures that none of the folders share a basename
func robocopyValidateBasenames(processedFolders [][]interface{}) error {

	basenameMap := map[string]interface{}{}
	for _, robocopyFolder := range processedFolders {

		robocopyFolderPath, ok := (robocopyFolder[0]).(string)
		if !ok {
			return fmt.Errorf("invalid robocopyFolderPath")
		}

		folderEntry, ok := (robocopyFolder[1]).(model.Folder)
		if !ok {
			return fmt.Errorf("invalid robocopyFolder")
		}

		// Use the name of the src folder as the dest folder name, unless
		// a replacement is specified in the folder entry.
		destFolderName := filepath.Base(robocopyFolderPath)
		if folderEntry.Robocopy != nil && folderEntry.Robocopy.DestFolderName != "" {
			destFolderName = folderEntry.Robocopy.DestFolderName
		}

		if _, contains := basenameMap[destFolderName]; contains {
			return fmt.Errorf("multiple folders share the same base name: %s", destFolderName)
		}

		basenameMap[destFolderName] = destFolderName
	}
	return nil
}
