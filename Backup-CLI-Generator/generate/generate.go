package generate

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jgwest/backup-cli/generic"
	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func RunGenerate(path string, outputPath string) error {

	model, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	result, err := ProcessConfig(path, model, false)
	if err != nil {
		return err
	}

	// If the output path already exists, don't overwrite it
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("output path already exists: %s", outputPath)
	}

	if err := ioutil.WriteFile(outputPath, []byte(result), 0700); err != nil {
		return err
	}

	fmt.Println(result)

	return nil

}

func backupPathContains(backupPaths []string, testStr string) bool {
	for _, backupPath := range backupPaths {
		if testStr == backupPath {
			return true
		}
	}
	return false
}

// checkMonitorFolders verifies that there are no unignored child folders of monitor folders.
func CheckMonitorFolders(configFilePath string, config model.ConfigFile) error {

	if len(config.MonitorFolders) == 0 {
		return nil
	}

	// Expand the folders to backup (ensuring they exist)
	expandedBackupPaths := []string{}
	for _, folder := range config.Folders {

		expandedPath, err := util.Expand(folder.Path, config.Substitutions)
		if err != nil {
			return err
		}

		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			return fmt.Errorf("'folders' path does not exist: '%s'", folder.Path)
		}

		expandedBackupPaths = append(expandedBackupPaths, expandedPath)
	}

	for _, monitorFolder := range config.MonitorFolders {

		monitorPath, err := util.Expand(monitorFolder.Path, config.Substitutions)
		if err != nil {
			return err
		}

		// If the paths to backup contain the monitor path itself, then we are good, so continue to the next item
		if backupPathContains(expandedBackupPaths, monitorPath) {
			continue
		}

		unbackedupPaths, err := findUnbackedUpPaths(monitorPath, monitorFolder, expandedBackupPaths)
		if err != nil {
			return err
		}

		if len(unbackedupPaths) != 0 {
			fmt.Println()
			fmt.Println("Un-backed-up paths found:")
			for _, ubPath := range unbackedupPaths {
				rel, err := filepath.Rel(monitorPath, ubPath)
				if err != nil {
					return err
				}
				fmt.Println("      - " + rel + "  # " + ubPath)
			}
			fmt.Println()

			return fmt.Errorf("monitor folder contained un-backed-up path: %v", unbackedupPaths)
		}

	}

	return nil
}

func findUnbackedUpPaths(monitorPath string, monitorFolder model.MonitorFolder, expandedBackupPaths []string) ([]string, error) {

	if _, err := os.Stat(monitorPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("'monitor path' does not exist: '%s' (%s)", monitorPath, monitorFolder.Path)
	}

	pathInfo, err := ioutil.ReadDir(monitorPath)
	if err != nil {
		return nil, err
	}

	unbackedupPaths := []string{}

	// For each child folder under the monitor folder...
outer:
	for _, monPathInfo := range pathInfo {
		if !monPathInfo.IsDir() {
			continue
		}

		fullPathName := filepath.Join(monitorPath, monPathInfo.Name())

		// For each of the excluded directories in the monitor folder, expand the glob if needed,
		// then see if the fullPathName is one of the glob matches; if so, skip it.
		for _, exclude := range monitorFolder.Excludes {

			globMatches, err := filepath.Glob(filepath.Join(monitorPath, exclude))
			if err != nil {
				return nil, err
			}

			// The folder from the folder list matched an exclude, so skip it
			if backupPathContains(globMatches, fullPathName) {
				continue outer
			}
		}

		// For each of the folders under monitorPath, ensure they are backed up
		if !backupPathContains(expandedBackupPaths, fullPathName) {
			unbackedupPaths = append(unbackedupPaths, fullPathName)
		}
	}

	return unbackedupPaths, nil

}

func ProcessConfig(configFilePath string, config model.ConfigFile, dryRun bool) (string, error) {

	configType, err := config.GetConfigType()
	if err != nil {
		return "", err
	}

	if dryRun && configType != model.Tarsnap {
		return "", fmt.Errorf("dryrun is only supported for tarsnap")
	}

	if err := CheckMonitorFolders(configFilePath, config); err != nil {
		return "", err
	}

	nodes := util.NewTextNodes()

	// buffer := util.OutputBuffer{
	// 	IsWindows: runtime.GOOS == "windows",
	// }

	// if buffer.IsWindows {
	// 	buffer.Lines = []string{"@echo off", "setlocal"}
	// 	// https://stackoverflow.com/questions/17063947/get-current-batchfile-directory
	// 	buffer.Out("set SCRIPTPATH=\"%~f0\"")
	// } else {
	// 	buffer.Lines = []string{"#!/bin/bash", "", "set -eu"}
	// 	// https://stackoverflow.com/questions/4774054/reliable-way-for-a-bash-script-to-get-the-full-path-to-itself
	// 	buffer.Out("SCRIPTPATH=`realpath -s $0`")
	// }

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

		if configType == model.Robocopy {
			return "", errors.New("robocopy does not support global excludes")
		}

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

			if configType == model.Kopia {
				// TODO: Kopia: This needs to be something different on Windows, probably without the slash
				excludesNode.SetEnv("EXCLUDES", substring+"--add-ignore \\\""+expandedValue+"\\\"")

				return "", fmt.Errorf("this needs to be something different on Windows, probably without the slash")

			} else if configType == model.Restic || configType == model.Tarsnap {
				if nodes.IsWindows() {
					excludesNode.SetEnv("EXCLUDES", substring+"--exclude \""+expandedValue+"\"")
				} else {
					excludesNode.SetEnv("EXCLUDES", substring+"--exclude \\\""+expandedValue+"\\\"")
				}
			}
		}
	}

	// Robocopy only: Populate EXCLUDES
	if config.RobocopySettings != nil {

		if configType != model.Robocopy || !nodes.IsWindows() {
			return "", errors.New("robocopy settings not supported for non-robocopy")
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
		processedFolders, err := PopulateProcessedFolders(configType, config.Folders, config.Substitutions, kopiaPolicyExcludes)
		if err != nil {
			return "", fmt.Errorf("unable to populateProcessedFolder: %v", err)
		}

		// Everything except robocopy
		if configType == model.Kopia || configType == model.Restic || configType == model.Tarsnap {
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
		} else if configType == model.Robocopy {

			// Ensure that none of the folders share a basename
			if err := RobocopyValidateBasenames(processedFolders); err != nil {
				return "", err
			}

			if robocopyCredentials, err := config.GetRobocopyCredential(); err == nil {

				robocopyFolders, err = RobocopyGenerateTargetPaths(processedFolders, robocopyCredentials)
				if err != nil {
					return "", err
				}

			} else {
				return "", err
			}

		} else { // end robocopy section
			return "", errors.New("unrecognized config type")
		}

	} // end 'process folders' section

	var invocationNode *util.TextNode

	if configType == model.Restic {

		// Uses the 'TODO' env var, generated above, to know what to backup.
		invocationNode, err = resticGenerateInvocation2(config, nodes)
		if err != nil {
			return "", err
		}

	} else if configType == model.Tarsnap {

		// Uses TODO, EXCLUDES, BACKUP_DATE_TIME, from above
		invocationNode, err = tarsnapGenerateInvocation2(config, dryRun, nodes)
		if err != nil {
			return "", err
		}

	} else if configType == model.Kopia {

		// Uses TODO, BACKUP_DATE_TIME, EXCLUDES, from above
		invocationNode, err = kopiaGenerateInvocation2(kopiaPolicyExcludes, config, nodes)
		if err != nil {
			return "", err
		}

	} else if configType == model.Robocopy {

		invocationNode, err = robocopyGenerateInvocation2(config, robocopyFolders, nodes)
		if err != nil {
			return "", err
		}

	} else {
		return "", errors.New("unsupported config")
	}

	suffixNode := nodes.NewTextNode()
	suffixNode.Out()
	suffixNode.Header("Verify the YAML file still produces this script")
	suffixNode.Out("backup-cli check \"" + configFilePath + "\" " + suffixNode.Env("SCRIPTPATH"))
	suffixNode.AddDependency(invocationNode)

	return nodes.ToString()
}

func robocopyGenerateInvocation2(config model.ConfigFile, robocopyFolders [][]string, textNodes *util.TextNodes) (*util.TextNode, error) {

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

// func robocopyGenerateInvocation(config model.ConfigFile, robocopyFolders [][]string, buffer *util.OutputBuffer) error {

// 	robocopyCredentials, err := config.GetRobocopyCredential()
// 	if err != nil {
// 		return err
// 	}

// 	if robocopyCredentials.DestinationFolder == "" {
// 		return errors.New("missing destination folder")
// 	}

// 	if robocopyCredentials.Switches == "" {
// 		return errors.New("missing switches")
// 	}

// 	if config.Metadata != nil && (config.Metadata.Name != "" || config.Metadata.AppendDateTime) {
// 		return fmt.Errorf("metadata features are not supported with robocopy")
// 	}

// 	if _, err := os.Stat(robocopyCredentials.DestinationFolder); os.IsNotExist(err) {
// 		return fmt.Errorf("robocopy destination folder does not exist: '%s'", robocopyCredentials.DestinationFolder)
// 	}

// 	buffer.SetEnv("SWITCHES", robocopyCredentials.Switches)

// 	for _, folderTuple := range robocopyFolders {
// 		srcFolder := util.FixWindowsPathSuffix("\"" + folderTuple[0] + "\"")
// 		destFolder := util.FixWindowsPathSuffix("\"" + folderTuple[1] + "\"")
// 		buffer.Out(fmt.Sprintf("robocopy %s %s %s", srcFolder, destFolder, buffer.Env("SWITCHES")))
// 	}

// 	return nil
// }

func kopiaGenerateInvocation2(kopiaPolicyExcludes map[string][]string, config model.ConfigFile, textNodes *util.TextNodes) (*util.TextNode, error) {

	textNode := textNodes.NewTextNode()

	kopiaCredentials, err := config.GetKopiaCredential()
	if err != nil {
		return nil, err
	}

	if kopiaCredentials.S3 == nil || kopiaCredentials.KopiaS3 == nil {
		return nil, fmt.Errorf("missing S3 credentials")
	}

	if kopiaCredentials.S3.AccessKeyID == "" || kopiaCredentials.S3.SecretAccessKey == "" {
		return nil, fmt.Errorf("missing S3 credential values")
	}

	if kopiaCredentials.KopiaS3.Bucket == "" || kopiaCredentials.KopiaS3.Region == "" || kopiaCredentials.KopiaS3.Endpoint == "" {
		return nil, fmt.Errorf("missing S3 credential values")
	}

	if kopiaCredentials.Password == "" {
		return nil, fmt.Errorf("missing kopia password")
	}

	credentialsNode := textNodes.NewTextNode()
	{

		credentialsNode.Out()
		credentialsNode.Header("Credentials ")
		credentialsNode.SetEnv("AWS_ACCESS_KEY_ID", kopiaCredentials.S3.AccessKeyID)
		credentialsNode.SetEnv("AWS_SECRET_ACCESS_KEY", kopiaCredentials.S3.SecretAccessKey)

		if len(kopiaCredentials.Password) > 0 {
			credentialsNode.SetEnv("KOPIA_PASSWORD", kopiaCredentials.Password)
		}
	}

	textNode.Out()
	textNode.Header("Connect repository")

	cliInvocation := fmt.Sprintf("kopia repository connect s3 --bucket=\"%s\" --access-key=\"%s\" --secret-access-key=\"%s\" --password=\"%s\" --endpoint=\"%s\" --region=\"%s\"",
		kopiaCredentials.KopiaS3.Bucket,
		textNode.Env("AWS_ACCESS_KEY_ID"),
		textNode.Env("AWS_SECRET_ACCESS_KEY"),
		textNode.Env("KOPIA_PASSWORD"),
		kopiaCredentials.KopiaS3.Endpoint,
		kopiaCredentials.KopiaS3.Region)

	textNode.Out(cliInvocation)

	if len(config.GlobalExcludes) > 0 {
		cliInvocation = fmt.Sprintf("kopia policy set --global %s", textNode.Env("EXCLUDES"))
		textNode.Out(cliInvocation)
	}

	// Build
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

// func kopiaGenerateInvocationOld(kopiaPolicyExcludes map[string][]string, config model.ConfigFile, buffer *util.OutputBuffer) error {

// 	kopiaCredentials, err := config.GetKopiaCredential()
// 	if err != nil {
// 		return err
// 	}

// 	if kopiaCredentials.S3 == nil || kopiaCredentials.KopiaS3 == nil {
// 		return fmt.Errorf("missing S3 credentials")
// 	}

// 	if kopiaCredentials.S3.AccessKeyID == "" || kopiaCredentials.S3.SecretAccessKey == "" {
// 		return fmt.Errorf("missing S3 credential values")
// 	}

// 	if kopiaCredentials.KopiaS3.Bucket == "" || kopiaCredentials.KopiaS3.Region == "" || kopiaCredentials.KopiaS3.Endpoint == "" {
// 		return fmt.Errorf("missing S3 credential values")
// 	}

// 	if kopiaCredentials.Password == "" {
// 		return fmt.Errorf("missing kopia password")
// 	}

// 	buffer.Out()
// 	buffer.Header("Credentials ")
// 	buffer.SetEnv("AWS_ACCESS_KEY_ID", kopiaCredentials.S3.AccessKeyID)
// 	buffer.SetEnv("AWS_SECRET_ACCESS_KEY", kopiaCredentials.S3.SecretAccessKey)

// 	if len(kopiaCredentials.Password) > 0 {
// 		buffer.SetEnv("KOPIA_PASSWORD", kopiaCredentials.Password)
// 	}

// 	buffer.Out()
// 	buffer.Header("Connect repository")

// 	cliInvocation := fmt.Sprintf("kopia repository connect s3 --bucket=\"%s\" --access-key=\"%s\" --secret-access-key=\"%s\" --password=\"%s\" --endpoint=\"%s\" --region=\"%s\"",
// 		kopiaCredentials.KopiaS3.Bucket,
// 		buffer.Env("AWS_ACCESS_KEY_ID"),
// 		buffer.Env("AWS_SECRET_ACCESS_KEY"),
// 		buffer.Env("KOPIA_PASSWORD"),
// 		kopiaCredentials.KopiaS3.Endpoint,
// 		kopiaCredentials.KopiaS3.Region)

// 	buffer.Out(cliInvocation)

// 	if len(config.GlobalExcludes) > 0 {
// 		cliInvocation = fmt.Sprintf("kopia policy set --global %s", buffer.Env("EXCLUDES"))
// 		buffer.Out(cliInvocation)
// 	}

// 	// Build
// 	if len(kopiaPolicyExcludes) > 0 {
// 		buffer.Out()
// 		buffer.Header("Add policy excludes")

// 		for backupPath, excludes := range kopiaPolicyExcludes {

// 			if len(excludes) == 0 {
// 				continue
// 			}

// 			excludesStr := ""
// 			for _, exclude := range excludes {
// 				excludesStr += "--add-ignore \"" + exclude + "\" "
// 			}
// 			excludesStr = strings.TrimSpace(excludesStr)

// 			cliInvocation = fmt.Sprintf("kopia policy set %s \"%s\"", excludesStr, backupPath)
// 			buffer.Out(cliInvocation)
// 		}
// 	}

// 	buffer.Out()
// 	buffer.Header("Create snapshot")

// 	descriptionSubstring := ""
// 	if config.Metadata != nil && config.Metadata.Name != "" {
// 		description := config.Metadata.Name

// 		if config.Metadata.AppendDateTime {
// 			description += buffer.Env("BACKUP_DATE_TIME")
// 		}

// 		descriptionSubstring = fmt.Sprintf("--description=\"%s\" ", description)
// 	}

// 	cliInvocation = fmt.Sprintf("kopia snapshot create %s%s",
// 		descriptionSubstring,
// 		buffer.Env("TODO"))

// 	if buffer.IsWindows {
// 		buffer.Out(cliInvocation)
// 	} else {
// 		buffer.Out("bash -c \"" + cliInvocation + "\"")
// 	}

// 	return nil
// }

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

// func tarsnapGenerateInvocationOld(config model.ConfigFile, dryRun bool, buffer *util.OutputBuffer) error {

// 	tarsnapCredentials, err := config.GetTarsnapCredential()
// 	if err != nil {
// 		return err
// 	}

// 	if _, err := os.Stat(tarsnapCredentials.ConfigFilePath); os.IsNotExist(err) {
// 		return fmt.Errorf("tarsnap config path does not exist: '%s'", tarsnapCredentials.ConfigFilePath)
// 	}

// 	if config.Metadata == nil || len(config.Metadata.Name) == 0 {
// 		return fmt.Errorf("tarsnap requires a metadata name")
// 	}

// 	backupName := config.Metadata.Name
// 	if config.Metadata.AppendDateTime {
// 		backupName += buffer.Env("BACKUP_DATE_TIME")
// 	}

// 	dryRunSubstring := ""
// 	if dryRun {
// 		dryRunSubstring = "--dry-run "
// 	}

// 	excludesSubstring := ""
// 	if len(config.GlobalExcludes) > 0 {
// 		excludesSubstring = buffer.Env("EXCLUDES") + " "
// 	}

// 	cliInvocation := fmt.Sprintf(
// 		"tarsnap --humanize-numbers --configfile \"%s\" -c %s%s -f \"%s\" %s",
// 		tarsnapCredentials.ConfigFilePath,
// 		dryRunSubstring,
// 		excludesSubstring,
// 		backupName,
// 		buffer.Env("TODO"))

// 	buffer.Out()

// 	if buffer.IsWindows {
// 		buffer.Out(cliInvocation)
// 	} else {
// 		buffer.Out("bash -c \"" + cliInvocation + "\"")
// 	}

// 	return nil
// }

func resticGenerateInvocation2(config model.ConfigFile, textNodes *util.TextNodes) (*util.TextNode, error) {

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

// func resticGenerateInvocation(config model.ConfigFile, buffer *util.OutputBuffer) error {

// 	textNodes := util.NewTextNodes()
// 	textNode := textNodes.NewTextNode()

// 	if err := generic.SharedGenerateResticCredentials(config, textNode); err != nil {
// 		return err
// 	}

// 	textNode.ExportTo(buffer)

// 	resticCredential, err := config.GetResticCredential()
// 	if err != nil {
// 		return err
// 	}

// 	{
// 		tagSubstring := ""
// 		if config.Metadata != nil {
// 			if len(config.Metadata.Name) == 0 {
// 				return errors.New("metadata exists, but name is nil")
// 			}

// 			quote := "'"
// 			if buffer.IsWindows {
// 				quote = "\""
// 			}

// 			tagSubstring = fmt.Sprintf("--tag %s%s", quote, config.Metadata.Name)
// 			if config.Metadata.AppendDateTime {
// 				tagSubstring += buffer.Env("BACKUP_DATE_TIME")
// 			}

// 			tagSubstring += quote + " "
// 		}

// 		url := ""
// 		if resticCredential.S3 != nil {
// 			url = "s3:" + resticCredential.S3.URL
// 		} else if resticCredential.RESTEndpoint != "" {
// 			url = "rest:" + resticCredential.RESTEndpoint
// 		} else {
// 			return errors.New("unable to locate connection credentials")
// 		}

// 		cacertSubstring := ""
// 		if resticCredential.CACert != "" {
// 			expandedPath, err := util.Expand(resticCredential.CACert, config.Substitutions)
// 			if err != nil {
// 				return err
// 			}
// 			cacertSubstring = "--cacert \"" + expandedPath + "\" "
// 		}

// 		excludesSubstring := ""
// 		if len(config.GlobalExcludes) > 0 {
// 			excludesSubstring = buffer.Env("EXCLUDES") + " "
// 		}

// 		cliInvocation := fmt.Sprintf("restic -r %s --verbose %s%s%s backup %s",
// 			url,
// 			tagSubstring,
// 			cacertSubstring,
// 			excludesSubstring,
// 			buffer.Env("TODO"))

// 		buffer.Out()

// 		if buffer.IsWindows {
// 			buffer.Out(cliInvocation)
// 		} else {
// 			buffer.Out("bash -c \"" + cliInvocation + "\"")
// 		}
// 	}

// 	return nil
// }

// RobocopyGenerateTargetPaths returns a slice of:
// - source folder path
// - destination folder (with basename of source folder appended)
// Example:
// - [C:\Users] -> [B:\backup\C-Users]
// - [D:\Users] -> [B:\backup\D-Users]
// - [C:\To-Backup] -> [B:\backup\To-Backup]
func RobocopyGenerateTargetPaths(processedFolders [][]interface{}, robocopyCredentials model.RobocopyCredentials) ([][]string, error) {
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

// RobocopyValidateBasenames ensures that none of the folders share a basename
func RobocopyValidateBasenames(processedFolders [][]interface{}) error {

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

// PopulateProcessedFolders performs error checking on config file folders, then returns
// the a tuple containing (folder path to backup, folder object)
func PopulateProcessedFolders(configType model.ConfigType, configFolders []model.Folder, configFileSubstitutions []model.Substitution, kopiaPolicyExcludes map[string][]string) ([][]interface{}, error) {

	var processedFolders [][]interface{}
	// Array of interfaces, containing:
	// - path of folder to backup
	// - the corresponding 'Folder' object

	// Populate processedFolders with list of folders to backup, and perform sanity tests
	checkDupesMap := map[string] /* source folder path -> not used */ interface{}{}
	for _, folder := range configFolders {

		// if len(folder.Excludes) != 0 &&
		// 	(configType == model.Restic ||
		// 		configType == model.Tarsnap ||
		// 		configType == model.Kopia ||
		// 		configType == model.Robocopy) {
		// 	return nil, fmt.Errorf("backup utility '%s' does not support local excludes", configType)
		// }

		if folder.Robocopy != nil && configType != model.Robocopy {
			return nil, fmt.Errorf("backup utility '%s' does not support robocopy folder entries", configType)
		}

		srcFolderPath, err := util.Expand(folder.Path, configFileSubstitutions)
		if err != nil {
			return nil, err
		}

		if _, err := os.Stat(srcFolderPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: '%s'", srcFolderPath)
		}

		if _, contains := checkDupesMap[srcFolderPath]; contains {
			return nil, fmt.Errorf("backup path list contains duplicate path: '%s'", srcFolderPath)
		}

		if len(folder.Excludes) != 0 &&
			(configType == model.Restic ||
				configType == model.Tarsnap ||
				configType == model.Robocopy) {
			return nil, fmt.Errorf("backup utility '%s' does not support local excludes", configType)

		} else if configType == model.Kopia {
			kopiaPolicyExcludes[srcFolderPath] = append(kopiaPolicyExcludes[srcFolderPath], folder.Excludes...)
		}

		processedFolders = append(processedFolders, []interface{}{srcFolderPath, folder})
	}

	return processedFolders, nil

}
