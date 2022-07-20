package generate

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jgwest/backup-cli/model"
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/yaml.v2"
)

func RunGenerate(path string, outputPath string) error {

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	// Look for invalid fields in the YAML
	if err := diffMissingFields(content); err != nil {
		return err
	}

	model := model.ConfigFile{}
	if err = yaml.Unmarshal(content, &model); err != nil {
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

	if err := ioutil.WriteFile(outputPath, []byte(result.ToString()), 0700); err != nil {
		return err
	}

	fmt.Println(result.ToString())

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
func checkMonitorFolders(configFilePath string, config model.ConfigFile) error {

	if len(config.MonitorFolders) == 0 {
		return nil
	}

	// Expand the folders to backup (ensuring they exist)
	expandedBackupPaths := []string{}
	for _, folder := range config.Folders {

		expandedPath, err := expand(folder.Path, config.Substitutions)
		if err != nil {
			return err
		}

		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			return fmt.Errorf("'folders' path does not exist: '%s'", folder.Path)
		}

		expandedBackupPaths = append(expandedBackupPaths, expandedPath)
	}

	// // return true if array contains exact string, false otherwise
	// contains := func(backupPaths []string, testStr string) bool {
	// 	for _, backupPath := range backupPaths {
	// 		if testStr == backupPath {
	// 			return true
	// 		}
	// 	}
	// 	return false
	// }

	for _, monitorFolder := range config.MonitorFolders {

		monitorPath, err := expand(monitorFolder.Path, config.Substitutions)
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

		// outer:
		// 	// For each child folder under the monitor folder...
		// 	for _, monPathInfo := range pathInfo {
		// 		if !monPathInfo.IsDir() {
		// 			continue
		// 		}

		// 		fullPathName := filepath.Join(monitorPath, monPathInfo.Name())

		// 		// For each of the excluded directories in the monitor folder, expand the glob if needed,
		// 		// then see if the fullPathName is one of the glob matches; if so, skip it.
		// 		for _, exclude := range monitorFolder.Excludes {

		// 			globMatches, err := filepath.Glob(filepath.Join(monitorPath, exclude))
		// 			if err != nil {
		// 				return err
		// 			}

		// 			// The folder from the folder list matched an exclude, so skip it
		// 			if backupPathContains(globMatches, fullPathName) {
		// 				continue outer
		// 			}
		// 		}

		// 		// For each of the folders under monitorPath, ensure they are backed up
		// 		if !backupPathContains(expandedBackupPaths, fullPathName) {
		// 			unbackedupPaths = append(unbackedupPaths, fullPathName)
		// 		}

		// 	}

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

func ProcessConfig(configFilePath string, config model.ConfigFile, dryRun bool) (*OutputBuffer, error) {

	configType, err := config.GetConfigType()
	if err != nil {
		return nil, err
	}

	if dryRun && configType != model.Tarsnap {
		return nil, fmt.Errorf("dryrun is only supported for tarsnap")
	}

	if err := checkMonitorFolders(configFilePath, config); err != nil {
		return nil, err
	}

	buffer := OutputBuffer{
		isWindows: runtime.GOOS == "windows",
	}

	if buffer.isWindows {
		buffer.lines = []string{"@echo off", "setlocal"}
		// https://stackoverflow.com/questions/17063947/get-current-batchfile-directory
		buffer.out("set SCRIPTPATH=\"%~f0\"")
	} else {
		buffer.lines = []string{"#!/bin/bash", "", "set -eu"}
		// https://stackoverflow.com/questions/4774054/reliable-way-for-a-bash-script-to-get-the-full-path-to-itself
		buffer.out("SCRIPTPATH=`realpath -s $0`")
	}

	if config.Metadata != nil {
		if config.Metadata.Name == "" {
			return nil, fmt.Errorf("if metadata is specified, then name must be specified")
		}

		if config.Metadata.AppendDateTime {
			if buffer.isWindows {
				buffer.out("set BACKUP_DATE_TIME=%DATE%-%TIME:~1%")
			} else {
				buffer.out("BACKUP_DATE_TIME=`date +%F_%H:%M:%S`")
			}
		}

	}

	// Populate EXCLUDES var, by processing Global Excludes
	if len(config.GlobalExcludes) > 0 {

		if configType == model.Robocopy {
			return nil, errors.New("robocopy does not support global excludes")
		}

		buffer.out()
		buffer.header("Excludes")
		for index, exclude := range config.GlobalExcludes {

			substring := ""

			if index > 0 {
				substring = buffer.env("EXCLUDES") + " "
			}

			expandedValue, err := expand(exclude, config.Substitutions)
			if err != nil {
				return nil, err
			}

			if configType == model.Kopia {
				// TODO: This needs to be something different on Windows, probably without the slash
				buffer.setEnv("EXCLUDES", substring+"--add-ignore \\\""+expandedValue+"\\\"")

				return nil, fmt.Errorf("this needs to be something different on Windows, probably without the slash")

			} else if configType == model.Restic || configType == model.Tarsnap {
				if buffer.isWindows {
					buffer.setEnv("EXCLUDES", substring+"--exclude \""+expandedValue+"\"")
				} else {
					buffer.setEnv("EXCLUDES", substring+"--exclude \\\""+expandedValue+"\\\"")
				}
			}
		}
	}

	// Robocopy only: Populate EXCLUDES
	if config.RobocopySettings != nil {

		if configType != model.Robocopy || !buffer.isWindows {
			return nil, errors.New("robocopy settings not supported for non-robocopy")
		}

		buffer.out()
		buffer.header("Excludes")

		excludesCount := 0

		for _, excludeFile := range config.RobocopySettings.ExcludeFiles {

			substring := ""

			if excludesCount > 0 {
				substring = buffer.env("EXCLUDES") + " "
			}

			expandedValue, err := expand(excludeFile, config.Substitutions)
			if err != nil {
				return nil, err
			}

			buffer.setEnv("EXCLUDES", substring+"/XF \""+expandedValue+"\"")

			excludesCount++
		}

		for _, excludeDir := range config.RobocopySettings.ExcludeFolders {

			substring := ""

			if excludesCount > 0 {
				substring = buffer.env("EXCLUDES") + " "
			}

			expandedValue, err := expand(excludeDir, config.Substitutions)
			if err != nil {
				return nil, err
			}

			if strings.Contains(expandedValue, "*") {
				return nil, fmt.Errorf("wildcards may not be supported in directories with robocopy: %s", expandedValue)
			}

			buffer.setEnv("EXCLUDES", substring+"/XD \""+expandedValue+"\"")

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
		if len(config.Folders) == 0 {
			return nil, errors.New("at least one folder is required")
		}

		buffer.out("")
		buffer.header("Folders")

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		// - This function also updates kopiaPolicyExcludes, if applicable.
		processedFolders, err := populateProcessedFolders(configType, config.Folders, config.Substitutions, kopiaPolicyExcludes)
		if err != nil {
			return nil, fmt.Errorf("unable to populateProcessedFolder: %v", err)
		}

		// // Populate processedFolders with list of folders to backup, and perform sanity tests
		// {
		// 	checkDupesMap := map[string] /* source folder path -> not used */ interface{}{}
		// 	for _, folder := range config.Folders {

		// 		if len(folder.Excludes) != 0 &&
		// 			(configType == model.Restic ||
		// 				configType == model.Tarsnap ||
		// 				configType == model.Kopia ||
		// 				configFilePath == model.Robocopy) {
		// 			return nil, fmt.Errorf("backup utility '%s' does not support local excludes", configType)
		// 		}

		// 		if folder.Robocopy != nil && configType != model.Robocopy {
		// 			return nil, fmt.Errorf("backup utility '%s' does not support robocopy folder entries", configType)
		// 		}

		// 		srcFolderPath, err := expand(folder.Path, config.Substitutions)
		// 		if err != nil {
		// 			return nil, err
		// 		}

		// 		if _, err := os.Stat(srcFolderPath); os.IsNotExist(err) {
		// 			return nil, fmt.Errorf("path does not exist: '%s'", srcFolderPath)
		// 		}

		// 		if _, contains := checkDupesMap[srcFolderPath]; contains {
		// 			return nil, fmt.Errorf("backup path list contains duplicate path: '%s'", srcFolderPath)
		// 		}

		// 		if len(folder.Excludes) != 0 &&
		// 			(configType == model.Restic ||
		// 				configType == model.Tarsnap ||
		// 				configType == model.Robocopy) {
		// 			return nil, fmt.Errorf("backup utility '%s' does not support local excludes", configType)

		// 		} else if configType == model.Kopia {
		// 			kopiaPolicyExcludes[srcFolderPath] = append(kopiaPolicyExcludes[srcFolderPath], folder.Excludes...)
		// 		}

		// 		processedFolders = append(processedFolders, []interface{}{srcFolderPath, folder})
		// 	}
		// }

		// Everything except robocopy
		if configType == model.Kopia || configType == model.Restic || configType == model.Tarsnap {
			for index, processedFolder := range processedFolders {
				substring := ""

				if index > 0 {
					substring = buffer.env("TODO") + " "
				}

				folderPath, ok := (processedFolder[0]).(string)
				if !ok {
					return nil, fmt.Errorf("invalid non-robocopy folderPath")
				}

				// TODO: This needs to be something different on Windows, probably without the slash

				// The unsubstituted path is used here

				if buffer.isWindows {
					buffer.setEnv("TODO", fmt.Sprintf("%s\"%s\"", substring, folderPath))
				} else {
					buffer.setEnv("TODO", fmt.Sprintf("%s\\\"%s\\\"", substring, folderPath))
				}

			}
		} else if configType == model.Robocopy {

			// Ensure that none of the folders share a basename
			if err := robocopyValidateBasenames(processedFolders); err != nil {
				return nil, err
			}
			//
			// {
			// 	basenameMap := map[string]string{}
			// 	for _, robocopyFolder := range processedFolders {

			// 		robocopyFolderPath, ok := (robocopyFolder[0]).(string)
			// 		if !ok {
			// 			return nil, fmt.Errorf("invalid robocopyFolderPath")
			// 		}

			// 		folderEntry, ok := (robocopyFolder[1]).(model.Folder)
			// 		if !ok {
			// 			return nil, fmt.Errorf("invalid robocopyFolder")
			// 		}

			// 		// Use the name of the src folder as the dest folder name, unless
			// 		// a replacement is specified in the folder entry.
			// 		destFolderName := filepath.Base(robocopyFolderPath)
			// 		if folderEntry.Robocopy != nil && folderEntry.Robocopy.DestFolderName != "" {
			// 			destFolderName = folderEntry.Robocopy.DestFolderName
			// 		}

			// 		if _, contains := basenameMap[destFolderName]; contains {
			// 			return nil, fmt.Errorf("multiple folders share the same base name: %s", destFolderName)
			// 		}

			// 		basenameMap[destFolderName] = destFolderName
			// 	}
			// }

			if robocopyCredentials, err := config.GetRobocopyCredential(); err == nil {

				robocopyFolders, err = robocopyGenerateTargetPaths(processedFolders, robocopyCredentials)
				if err != nil {
					return nil, err
				}

			} else {
				return nil, err
			}

			// targetFolder := robocopyCredentials.DestinationFolder

			// for _, robocopyFolder := range processedFolders {

			// 	robocopySrcFolderPath, ok := (robocopyFolder[0]).(string)
			// 	if !ok {
			// 		return nil, fmt.Errorf("invalid robocopyFolderPath")
			// 	}

			// 	folderEntry, ok := (robocopyFolder[1]).(model.Folder)
			// 	if !ok {
			// 		return nil, fmt.Errorf("invalid robocopyFolder")
			// 	}

			// 	// Use the name of the src folder as the dest folder name, unless
			// 	// a replacement is specified in the folder entry.
			// 	destFolderName := filepath.Base(robocopySrcFolderPath)
			// 	if folderEntry.Robocopy != nil && folderEntry.Robocopy.DestFolderName != "" {
			// 		destFolderName = folderEntry.Robocopy.DestFolderName
			// 	}

			// 	// tuple:
			// 	// - source folder path
			// 	// - destination folder with basename of source folder appended
			// 	tuple := []string{robocopySrcFolderPath, filepath.Join(targetFolder, destFolderName)}

			// 	// fmt.Println("- ["+robocopySrcFolderPath+"]", "["+filepath.Join(targetFolder, destFolderName)+"]")

			// 	robocopyFolders = append(robocopyFolders, tuple)

			// }

		} else { // end robocopy section
			return nil, errors.New("unrecognized config type")
		}

	} // end 'process folders' section

	if configType == model.Restic {

		// Uses the 'TODO' env var, generated above, to know what to backup.
		err = resticGenerateInvocation(config, &buffer)
		if err != nil {
			return nil, err
		}

	} else if configType == model.Tarsnap {

		// Uses TODO, EXCLUDES, BACKUP_DATE_TIME, from above
		err := tarsnapGenerateInvocation(config, dryRun, &buffer)
		if err != nil {
			return nil, err
		}

	} else if configType == model.Kopia {

		// Uses TODO, BACKUP_DATE_TIME, EXCLUDES, from above
		err = kopiaGenerateInvocation(kopiaPolicyExcludes, config, &buffer)
		if err != nil {
			return nil, err
		}

	} else if configType == model.Robocopy {

		err := robocopyGenerateInvocation(config, robocopyFolders, &buffer)
		if err != nil {
			return nil, err
		}

	} else {
		return nil, errors.New("unsupported config")
	}

	buffer.out()
	buffer.header("Verify the YAML file still produces this script")
	buffer.out("backup-cli check \"" + configFilePath + "\" " + buffer.env("SCRIPTPATH"))

	return &buffer, nil
}

func robocopyGenerateInvocation(config model.ConfigFile, robocopyFolders [][]string, buffer *OutputBuffer) error {

	robocopyCredentials, err := config.GetRobocopyCredential()
	if err != nil {
		return err
	}

	if robocopyCredentials.DestinationFolder == "" {
		return errors.New("missing destination folder")
	}

	if robocopyCredentials.Switches == "" {
		return errors.New("missing switches")
	}

	if config.Metadata != nil && (config.Metadata.Name != "" || config.Metadata.AppendDateTime) {
		return fmt.Errorf("metadata features are not supported with robocopy")
	}

	if _, err := os.Stat(robocopyCredentials.DestinationFolder); os.IsNotExist(err) {
		return fmt.Errorf("robocopy destination folder does not exist: '%s'", robocopyCredentials.DestinationFolder)
	}

	buffer.setEnv("SWITCHES", robocopyCredentials.Switches)

	for _, folderTuple := range robocopyFolders {
		srcFolder := fixWindowsPathSuffix("\"" + folderTuple[0] + "\"")
		destFolder := fixWindowsPathSuffix("\"" + folderTuple[1] + "\"")
		buffer.out(fmt.Sprintf("robocopy %s %s %s", srcFolder, destFolder, buffer.env("SWITCHES")))
	}

	return nil
}

func kopiaGenerateInvocation(kopiaPolicyExcludes map[string][]string, config model.ConfigFile, buffer *OutputBuffer) error {

	kopiaCredentials, err := config.GetKopiaCredential()
	if err != nil {
		return err
	}

	if kopiaCredentials.S3 == nil || kopiaCredentials.KopiaS3 == nil {
		return fmt.Errorf("missing S3 credentials")
	}

	if kopiaCredentials.S3.AccessKeyID == "" || kopiaCredentials.S3.SecretAccessKey == "" {
		return fmt.Errorf("missing S3 credential values")
	}

	if kopiaCredentials.KopiaS3.Bucket == "" || kopiaCredentials.KopiaS3.Region == "" || kopiaCredentials.KopiaS3.Endpoint == "" {
		return fmt.Errorf("missing S3 credential values")
	}

	if kopiaCredentials.Password == "" {
		return fmt.Errorf("missing kopia password")
	}

	buffer.out()
	buffer.header("Credentials ")
	buffer.setEnv("AWS_ACCESS_KEY_ID", kopiaCredentials.S3.AccessKeyID)
	buffer.setEnv("AWS_SECRET_ACCESS_KEY", kopiaCredentials.S3.SecretAccessKey)

	if len(kopiaCredentials.Password) > 0 {
		buffer.setEnv("KOPIA_PASSWORD", kopiaCredentials.Password)
	}

	buffer.out()
	buffer.header("Connect repository")

	cliInvocation := fmt.Sprintf("kopia repository connect s3 --bucket=\"%s\" --access-key=\"%s\" --secret-access-key=\"%s\" --password=\"%s\" --endpoint=\"%s\" --region=\"%s\"",
		kopiaCredentials.KopiaS3.Bucket,
		buffer.env("AWS_ACCESS_KEY_ID"),
		buffer.env("AWS_SECRET_ACCESS_KEY"),
		buffer.env("KOPIA_PASSWORD"),
		kopiaCredentials.KopiaS3.Endpoint,
		kopiaCredentials.KopiaS3.Region)

	buffer.out(cliInvocation)

	if len(config.GlobalExcludes) > 0 {
		cliInvocation = fmt.Sprintf("kopia policy set --global %s", buffer.env("EXCLUDES"))
		buffer.out(cliInvocation)
	}

	// Build
	if len(kopiaPolicyExcludes) > 0 {
		buffer.out()
		buffer.header("Add policy excludes")

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
			buffer.out(cliInvocation)
		}
	}

	buffer.out()
	buffer.header("Create snapshot")

	descriptionSubstring := ""
	if config.Metadata != nil && config.Metadata.Name != "" {
		description := config.Metadata.Name

		if config.Metadata.AppendDateTime {
			description += buffer.env("BACKUP_DATE_TIME")
		}

		descriptionSubstring = fmt.Sprintf("--description=\"%s\" ", description)
	}

	cliInvocation = fmt.Sprintf("kopia snapshot create %s%s",
		descriptionSubstring,
		buffer.env("TODO"))

	if buffer.isWindows {
		buffer.out(cliInvocation)
	} else {
		buffer.out("bash -c \"" + cliInvocation + "\"")
	}

	return nil
}

func tarsnapGenerateInvocation(config model.ConfigFile, dryRun bool, buffer *OutputBuffer) error {

	tarsnapCredentials, err := config.GetTarsnapCredential()
	if err != nil {
		return err
	}

	if _, err := os.Stat(tarsnapCredentials.ConfigFilePath); os.IsNotExist(err) {
		return fmt.Errorf("tarsnap config path does not exist: '%s'", tarsnapCredentials.ConfigFilePath)
	}

	if config.Metadata == nil || len(config.Metadata.Name) == 0 {
		return fmt.Errorf("tarsnap requires a metadata name")
	}

	backupName := config.Metadata.Name
	if config.Metadata.AppendDateTime {
		backupName += buffer.env("BACKUP_DATE_TIME")
	}

	dryRunSubstring := ""
	if dryRun {
		dryRunSubstring = "--dry-run "
	}

	excludesSubstring := ""
	if len(config.GlobalExcludes) > 0 {
		excludesSubstring = buffer.env("EXCLUDES") + " "
	}

	cliInvocation := fmt.Sprintf(
		"tarsnap --humanize-numbers --configfile \"%s\" -c %s%s -f \"%s\" %s",
		tarsnapCredentials.ConfigFilePath,
		dryRunSubstring,
		excludesSubstring,
		backupName,
		buffer.env("TODO"))

	buffer.out()

	if buffer.isWindows {
		buffer.out(cliInvocation)
	} else {
		buffer.out("bash -c \"" + cliInvocation + "\"")
	}

	return nil
}

func resticGenerateInvocation(config model.ConfigFile, buffer *OutputBuffer) error {

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return err
	}

	if resticCredential.S3 != nil {
		buffer.out()
		buffer.header("Credentials ")
		buffer.setEnv("AWS_ACCESS_KEY_ID", resticCredential.S3.AccessKeyID)
		buffer.setEnv("AWS_SECRET_ACCESS_KEY", resticCredential.S3.SecretAccessKey)
	}

	if len(resticCredential.Password) > 0 && len(resticCredential.PasswordFile) > 0 {
		return errors.New("both password and password file are specified")
	}

	if len(resticCredential.Password) > 0 {
		buffer.setEnv("RESTIC_PASSWORD", resticCredential.Password)

	} else if len(resticCredential.PasswordFile) > 0 {
		buffer.setEnv("RESTIC_PASSWORD_FILE", resticCredential.PasswordFile)

	} else {
		return errors.New("no restic password found")
	}

	tagSubstring := ""
	if config.Metadata != nil {
		if len(config.Metadata.Name) == 0 {
			return errors.New("metadata exists, but name is nil")
		}

		quote := "'"
		if buffer.isWindows {
			quote = "\""
		}

		tagSubstring = fmt.Sprintf("--tag %s%s", quote, config.Metadata.Name)
		if config.Metadata.AppendDateTime {
			tagSubstring += buffer.env("BACKUP_DATE_TIME")
		}

		tagSubstring += quote + " "
	}

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
		expandedPath, err := expand(resticCredential.CACert, config.Substitutions)
		if err != nil {
			return err
		}
		cacertSubstring = "--cacert \"" + expandedPath + "\" "
	}

	excludesSubstring := ""
	if len(config.GlobalExcludes) > 0 {
		excludesSubstring = buffer.env("EXCLUDES") + " "
	}

	cliInvocation := fmt.Sprintf("restic -r %s --verbose %s%s%s backup %s",
		url,
		tagSubstring,
		cacertSubstring,
		excludesSubstring,
		buffer.env("TODO"))

	buffer.out()

	if buffer.isWindows {
		buffer.out(cliInvocation)
	} else {
		buffer.out("bash -c \"" + cliInvocation + "\"")
	}

	return nil
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

type OutputBuffer struct {
	isWindows bool
	lines     []string
}

// populateProcessedFolders performs error checking on config file folders, then returns
// the a tuple containing (folder path to backup, folder object)
func populateProcessedFolders(configType model.ConfigType, configFolders []model.Folder, configFileSubstitutions []model.Substitution, kopiaPolicyExcludes map[string][]string) ([][]interface{}, error) {

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

		srcFolderPath, err := expand(folder.Path, configFileSubstitutions)
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

// expand returns the input string, replacing $var with config file substitutions, or env vars, in that order.
func expand(input string, configFileSubstitutions []model.Substitution) (output string, err error) {

	substitutions := map[string]string{}

	for _, substitution := range configFileSubstitutions {
		substitutions[substitution.Name] = substitution.Value
	}

	output = os.Expand(input, func(key string) string {

		if val, contains := substitutions[key]; contains {
			return val
		}

		if value, contains := os.LookupEnv(key); contains {
			return value
		}

		if err == nil {
			err = fmt.Errorf("unable to find value for '%s'", key)
		}

		return ""

	})

	return
}

func (buffer *OutputBuffer) ToString() string {
	output := ""

	for _, line := range buffer.lines {

		output += line

		if buffer.isWindows {
			output += "\r\n"
		} else {
			output += "\n"
		}
	}

	return output
}

func fixWindowsPathSuffix(input string) string {

	if strings.HasSuffix(input, "\\\"") {
		input = input[0 : len(input)-2]
		input += "\\\\\""
	}
	return input
}

func (buffer *OutputBuffer) setEnv(envName string, value string) {
	if buffer.isWindows {

		value = fixWindowsPathSuffix(value)

		// convert "Z:\" to "Z:\\"
		// if strings.HasSuffix(value, "\\\"") {
		// 	value = value[0 : len(value)-2]
		// 	value += "\\\\\""
		// }

		buffer.out(fmt.Sprintf("set %s=%s", envName, value))
	} else {
		// Export is used due to need to use 'bash -c (...)' at end of script
		buffer.out(fmt.Sprintf("export %s=\"%s\"", envName, value))
	}
}

func (buffer *OutputBuffer) env(envName string) string {
	if buffer.isWindows {
		return "%" + envName + "%"
	} else {
		return "${" + envName + "}"
	}
}

func (buffer *OutputBuffer) header(str string) {

	if !strings.HasSuffix(str, " ") {
		str += " "
	}

	for len(str) < 80 {
		str = str + "-"
	}

	if buffer.isWindows {
		buffer.out("REM " + str)
	} else {
		buffer.out("# " + str)
	}

}

// func (buffer *OutputBuffer) comment(str string) {
// 	if buffer.isWindows {
// 		buffer.out("REM " + str)
// 	} else {
// 		buffer.out("# " + str)
// 	}

// }

func (buffer *OutputBuffer) out(str ...string) {
	if len(str) == 0 {
		str = []string{""}
	}

	buffer.lines = append(buffer.lines, str...)
}

func diffMissingFields(content []byte) (err error) {

	convertToInterfaceAndBack := func(content []byte) (mapString string, err error) {

		// Convert to string => interface
		mapStringToIntr := map[string]interface{}{}
		if err = yaml.Unmarshal(content, &mapStringToIntr); err != nil {
			return
		}

		// Convert back to string
		var out []byte
		if out, err = yaml.Marshal(mapStringToIntr); err != nil {
			return
		}
		mapString = string(out)

		return
	}

	var mapString string
	if mapString, err = convertToInterfaceAndBack(content); err != nil {
		return
	}

	var structString string
	{
		// Convert string -> ConfigFile
		model := model.ConfigFile{}
		if err = yaml.Unmarshal(content, &model); err != nil {
			return
		}

		// Convert ConfigFile -> string
		var out []byte
		if out, err = yaml.Marshal(model); err != nil {
			return
		}
		if structString, err = convertToInterfaceAndBack(out); err != nil {
			return
		}
	}

	// Compare the two
	{
		dmp := diffmatchpatch.New()

		diffs := dmp.DiffMain(mapString, structString, false)

		nonequalDiffs := []diffmatchpatch.Diff{}

		for index, currDiff := range diffs {
			if currDiff.Type != diffmatchpatch.DiffEqual {
				nonequalDiffs = append(nonequalDiffs, diffs[index])
			}
		}

		if len(nonequalDiffs) > 0 {

			fmt.Println()
			fmt.Println("-------")
			fmt.Println(dmp.DiffPrettyText(diffs))
			fmt.Println("-------")
			return errors.New("diffs reported")
		}
	}

	return nil
}
