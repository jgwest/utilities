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

func checkMonitorFolders(configFilePath string, config model.ConfigFile) error {

	if len(config.MonitorFolders) == 0 {
		return nil
	}

	expandedBackupPaths := []string{}

	for _, folder := range config.Folders {

		expandedPath, err := expand(folder.Path, config)
		if err != nil {
			return err
		}

		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			return fmt.Errorf("'folders' path does not exist: '%s'", folder.Path)
		}

		expandedBackupPaths = append(expandedBackupPaths, expandedPath)
	}

	// return true if array contains string, false otherwise
	contains := func(backupPaths []string, testStr string) bool {
		for _, backupPath := range backupPaths {
			if testStr == backupPath {
				return true
			}
		}
		return false
	}

	for _, monitorFolder := range config.MonitorFolders {

		monitorPath, err := expand(monitorFolder.Path, config)
		if err != nil {
			return err
		}

		// If the paths to backup contain the monitor path itself, then we are good, so continue to the next item
		if contains(expandedBackupPaths, monitorPath) {
			continue
		}

		if _, err := os.Stat(monitorPath); os.IsNotExist(err) {
			return fmt.Errorf("'monitor path' does not exist: '%s' (%s)", monitorPath, monitorFolder.Path)
		}

		pathInfo, err := ioutil.ReadDir(monitorPath)
		if err != nil {
			return err
		}

		unbackedupPaths := []string{}

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
					return err
				}

				// The folder from the folder list matched an exclude, so skip it
				if contains(globMatches, fullPathName) {
					continue outer
				}
			}

			// For each of the folders under monitorPath, ensure they are backed up
			if !contains(expandedBackupPaths, fullPathName) {
				unbackedupPaths = append(unbackedupPaths, fullPathName)
			}

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

	// Process Global Excludes
	if len(config.GlobalExcludes) > 0 {

		if configType == model.Robocopy {
			return nil, errors.New("robocopy does not support excludes")
		}

		buffer.out()
		buffer.header("Excludes")
		for index, exclude := range config.GlobalExcludes {

			substring := ""

			if index > 0 {
				substring = buffer.env("EXCLUDES") + " "
			}

			expandedValue, err := expand(exclude, config)
			if err != nil {
				return nil, err
			}

			if configType == model.Kopia {
				// TODO: This needs to be something different on Windows, probably without the slash
				buffer.setEnv("EXCLUDES", substring+"--add-ignore \\\""+expandedValue+"\\\"")

			} else if configType == model.Restic || configType == model.Tarsnap {
				if buffer.isWindows {
					buffer.setEnv("EXCLUDES", substring+"--exclude \""+expandedValue+"\"")
				} else {
					buffer.setEnv("EXCLUDES", substring+"--exclude \\\""+expandedValue+"\\\"")
				}
			}
		}
	}

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

			expandedValue, err := expand(excludeFile, config)
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

			expandedValue, err := expand(excludeDir, config)
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
	// - destination folder with basename of source folder appended
	var robocopyFolders [][]string

	// Process folders
	// - Populate TODO env var, for everything except robocopy
	// - For robocopy, populate robocopyFolders
	{
		if len(config.Folders) == 0 {
			return nil, errors.New("at least one folder is required")
		}

		buffer.out("")
		buffer.header("Folders")

		// slice of [string, model.Folder{}]
		var processedFolders [][]interface{}

		// Populate processedFolders with list of folders to backup, and perform sanity tests
		{
			checkDupesMap := map[string]string{}
			for _, folder := range config.Folders {

				if len(folder.Excludes) != 0 &&
					(configType == model.Restic ||
						configType == model.Tarsnap ||
						configType == model.Kopia ||
						configFilePath == model.Robocopy) {
					return nil, fmt.Errorf("backup utility '%s' does not support local excludes", configType)
				}

				if folder.Robocopy != nil && configType != model.Robocopy {
					return nil, fmt.Errorf("backup utility '%s' does not support robocopy folder entries", configType)
				}

				srcFolderPath, err := expand(folder.Path, config)
				if err != nil {
					return nil, err
				}

				if _, err := os.Stat(srcFolderPath); os.IsNotExist(err) {
					return nil, fmt.Errorf("path does not exist: '%s'", srcFolderPath)
				}

				if _, contains := checkDupesMap[srcFolderPath]; contains {
					return nil, fmt.Errorf("backup path list contains duplicate path: '%s'", srcFolderPath)
				}

				processedFolders = append(processedFolders, []interface{}{srcFolderPath, folder})
			}
		}

		// Everything except robocopy
		if configType == model.Kopia || configType == model.Restic || configType == model.Tarsnap {
			for index, processedFolder := range processedFolders {
				substring := ""

				if index > 0 {
					substring = buffer.env("TODO") + " "
				}

				folderPath, ok := (processedFolder[0]).(string)
				if !ok {
					return nil, fmt.Errorf("invalid robocopyFolderPath")
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

			// TODO: Write a test for this:

			// Ensure that none of the folders share a basename
			{
				basenameMap := map[string]string{}
				for _, robocopyFolder := range processedFolders {

					robocopyFolderPath, ok := (robocopyFolder[0]).(string)
					if !ok {
						return nil, fmt.Errorf("invalid robocopyFolderPath")
					}

					folderEntry, ok := (robocopyFolder[1]).(model.Folder)
					if !ok {
						return nil, fmt.Errorf("invalid robocopyFolder")
					}

					// Use the name of the src folder as the dest folder name, unless
					// a replacement is specified in the folder entry.
					destFolderName := filepath.Base(robocopyFolderPath)
					if folderEntry.Robocopy != nil && folderEntry.Robocopy.DestFolderName != "" {
						destFolderName = folderEntry.Robocopy.DestFolderName
					}

					if _, contains := basenameMap[destFolderName]; contains {
						return nil, fmt.Errorf("multiple folders share the same base name: %s", destFolderName)
					}

					basenameMap[destFolderName] = destFolderName
				}
			}

			robocopyCredentials, err := config.GetRobocopyCredential()
			if err != nil {
				return nil, err
			}

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
				robocopyFolders = append(robocopyFolders, tuple)

			}

		} else { // end robocopy section
			return nil, errors.New("unrecognized config")
		}

	} // end 'process folders' section

	if configType == model.Restic {
		resticCredential, err := config.GetResticCredential()
		if err != nil {
			return nil, err
		}

		if resticCredential.S3 != nil {
			buffer.out()
			buffer.header("Credentials ")
			buffer.setEnv("AWS_ACCESS_KEY_ID", resticCredential.S3.AccessKeyID)
			buffer.setEnv("AWS_SECRET_ACCESS_KEY", resticCredential.S3.SecretAccessKey)
		}

		if len(resticCredential.Password) > 0 && len(resticCredential.PasswordFile) > 0 {
			return nil, errors.New("both password and password file are specified")
		}

		if len(resticCredential.Password) > 0 {
			buffer.setEnv("RESTIC_PASSWORD", resticCredential.Password)

		} else if len(resticCredential.PasswordFile) > 0 {
			buffer.setEnv("RESTIC_PASSWORD_FILE", resticCredential.PasswordFile)

		} else {
			return nil, errors.New("no restic password found")
		}

		tagSubstring := ""
		if config.Metadata != nil {

			if len(config.Metadata.Name) == 0 {
				return nil, errors.New("metadata exists, but name is nil")
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
			return nil, errors.New("unable to locate connection credentials")
		}

		cacertSubstring := ""
		if resticCredential.CACert != "" {
			expandedPath, err := expand(resticCredential.CACert, config)
			if err != nil {
				return nil, err
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

	} else if configType == model.Tarsnap {

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

		cliInvocation := fmt.Sprintf("tarsnap --humanize-numbers --configfile \"%s\" -c %s%s -f \"%s\" %s",
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

		// buffer.out(cliInvocation)

	} else if configType == model.Kopia {

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

		// buffer.out(cliInvocation)

	} else if configType == model.Robocopy {

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

		buffer.setEnv("SWITCHES", robocopyCredentials.Switches)

		for _, folderTuple := range robocopyFolders {
			buffer.out(fmt.Sprintf("robocopy \"%s\" \"%s\" %s", folderTuple[0], folderTuple[1], buffer.env("SWITCHES")))
		}

	} else {
		return nil, errors.New("unsupported config")
	}

	buffer.out()
	buffer.header("Verify the YAML file still produces this script")
	buffer.out("backup-cli check \"" + configFilePath + "\" " + buffer.env("SCRIPTPATH"))

	return &buffer, nil
}

type OutputBuffer struct {
	isWindows bool
	lines     []string
}

func expand(input string, configFile model.ConfigFile) (output string, err error) {

	substitutions := map[string]string{}

	for _, substitution := range configFile.Substitutions {
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

func (buffer *OutputBuffer) setEnv(envName string, value string) {
	if buffer.isWindows {

		// convert "Z:\" to "Z:\\"
		if strings.HasSuffix(value, "\\\"") {
			value = value[0 : len(value)-2]
			value += "\\\\\""
		}

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
