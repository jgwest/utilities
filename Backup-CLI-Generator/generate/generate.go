package generate

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	err = yaml.Unmarshal(content, &model)
	if err != nil {
		return err
	}

	result, err := ProcessConfig(path, model, false)
	if err != nil {
		return err
	}

	// TODO: Generate then diff

	// TODO: Add robocopy

	// if err := ioutil.WriteFile(outputPath, []byte(result.ToString()), 0600); err != nil {
	// 	return err
	// }

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

		monitorPath := os.ExpandEnv(monitorFolder.Path)

		// If the paths to backup contain the monitor path itself, then we are good, so continue to the next item
		if contains(expandedBackupPaths, monitorPath) {
			continue
		}

		if _, err := os.Stat(monitorPath); os.IsNotExist(err) {
			return fmt.Errorf("'monitor path' does not exist: '%s'", monitorPath)
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
				fmt.Println("      - " + rel)
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
		isWindows: false,
	}

	if buffer.isWindows {
		buffer.lines = []string{"@echo off", "setlocal"}
		// https://stackoverflow.com/questions/17063947/get-current-batchfile-directory
		buffer.out("set SCRIPTPATH=\"%~f0\"")
	} else {
		buffer.lines = []string{"#!/bin/bash", "", "set -eu"}
		// https://stackoverflow.com/questions/4774054/reliable-way-for-a-bash-script-to-get-the-full-path-to-itself
		buffer.out("SCRIPTPATH=\"$( cd -- \"$(dirname \"$0\")\" >/dev/null 2>&1 ; pwd -P )\"")
	}

	if config.Metadata != nil && config.Metadata.AppendDateTime {
		if buffer.isWindows {
			buffer.out("set BACKUP_DATE_TIME=%DATE%-%TIME:~1%")
		} else {
			buffer.out("BACKUP_DATE_TIME=`date +%F_%H:%M:%S`")
		}
	}

	buffer.out()
	buffer.comment("Verify the YAML file still produces this script")
	buffer.out("backup-cli check \"" + configFilePath + "\" " + buffer.env("SCRIPTPATH"))

	// \/ \/ \/ \/

	// TODO: WARNING - THIS WILL BREAK CRONTAB BACKUPS!!!!!!!!!!!!!!!!!!!!!!

	// /\ /\ /\ /\

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
				// substring = "$EXCLUDES "
				substring = buffer.env("EXCLUDES") + " "
			}

			expandedValue, err := expand(exclude, config)
			if err != nil {
				return nil, err
			}

			if configType == model.Kopia {
				buffer.setEnv("EXCLUDES", substring+"--add-ignore '"+expandedValue+"'")

			} else if configType == model.Restic || configType == model.Tarsnap {
				buffer.setEnv("EXCLUDES", substring+"--exclude '"+expandedValue+"'")
				// buffer.out("EXCLUDES=\"" + substring + "--exclude '" + exclude + "'\"")
			}

		}
	}

	var robocopyFolders [][]string

	// Process folders
	{
		if len(config.Folders) == 0 {
			return nil, errors.New("at least one folder is required")
		}

		buffer.out("")
		buffer.header("Folders")

		var processedFolders [][]interface{}

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

			folderPath, err := expand(folder.Path, config)
			if err != nil {
				return nil, err
			}

			if _, err := os.Stat(folderPath); os.IsNotExist(err) {
				return nil, fmt.Errorf("path does not exist: '%s'", folderPath)
			}

			processedFolders = append(processedFolders, []interface{}{folderPath, folder})

			// substring := ""

			// if index > 0 {
			// 	substring = buffer.env("TODO") + " "
			// }

			// // The unsubstituted path is used here
			// buffer.setEnv("TODO", fmt.Sprintf("%s'%s'", substring, folderPath))
		}

		if configType == model.Kopia || configType == model.Restic || configType == model.Tarsnap {
			for index, folderPath := range processedFolders {
				substring := ""

				if index > 0 {
					substring = buffer.env("TODO") + " "
				}

				// The unsubstituted path is used here
				buffer.setEnv("TODO", fmt.Sprintf("%s'%s'", substring, folderPath))
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

					key := filepath.Base(robocopyFolderPath)
					if folderEntry.Robocopy != nil && folderEntry.Robocopy.DestFolderName != "" {
						key = folderEntry.Robocopy.DestFolderName
					}

					if _, contains := basenameMap[key]; contains {
						return nil, fmt.Errorf("multiple folders share the same base name: %s", key)
					}

					basenameMap[key] = key
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

				destDirName := filepath.Base(robocopySrcFolderPath)
				if folderEntry.Robocopy != nil && folderEntry.Robocopy.DestFolderName != "" {
					destDirName = folderEntry.Robocopy.DestFolderName
				}

				// tuple:
				// - source folder path
				// - destination folder with basename of source folder appended
				tuple := []string{robocopySrcFolderPath, filepath.Join(targetFolder, destDirName)}
				robocopyFolders = append(robocopyFolders, tuple)

			}

			// for _, robocopyFolder := range processedFolders {
			// 	// tuple:
			// 	// - source folder path
			// 	// - destination folder with basename of source folder appended
			// 	tuple := []string{robocopyFolder, filepath.Join(targetFolder, filepath.Base(robocopyFolder))}
			// 	robocopyFolders = append(robocopyFolders, tuple)

			// }

		} else {
			return nil, errors.New("unrecognized config")
		}

	}

	// TODO: For all of these, ensure there are no duplicates in the backup paths

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
			// buffer.out(fmt.Sprintf("AWS_ACCESS_KEY_ID=\"%s\"", resticCredential.S3.AccessKeyID))
			// buffer.out(fmt.Sprintf("AWS_SECRET_ACCESS_KEY=\"%s\"", resticCredential.S3.SecretAccessKey))
		}

		if len(resticCredential.Password) > 0 && len(resticCredential.PasswordFile) > 0 {
			return nil, errors.New("both password and password file are specified")
		}

		if len(resticCredential.Password) > 0 {
			buffer.setEnv("RESTIC_PASSWORD", resticCredential.Password)
			// buffer.out(fmt.Sprintf("RESTIC_PASSWORD=\"%s\"", resticCredential.Password))

		} else if len(resticCredential.Password) > 0 {
			buffer.setEnv("RESTIC_PASSWORD_FILE", resticCredential.PasswordFile)
			// buffer.out(fmt.Sprintf("RESTIC_PASSWORD_FILE=\"%s\"", resticCredential.PasswordFile))

		} else {
			return nil, errors.New("no restic password found")
		}

		tagSubstring := ""

		if config.Metadata != nil {

			if len(config.Metadata.Name) == 0 {
				return nil, errors.New("metadata exists, but name is nil")
			}

			tagSubstring = fmt.Sprintf("--tag \"%s", config.Metadata.Name)
			if config.Metadata.AppendDateTime {
				// buffer.out()
				// if buffer.isWindows {
				// 	buffer.out("set BACKUP_DATE_TIME=%DATE%-%TIME:~1%")
				// } else {
				// 	buffer.out("BACKUP_DATE_TIME=`date +%F_%H:%M:%S`")
				// }

				tagSubstring += buffer.env("BACKUP_DATE_TIME")
			}

			tagSubstring += "\" "
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

		cliInvocation := fmt.Sprintf("restic -r %s --verbose %s%s%s backup %s",
			url,
			tagSubstring,
			cacertSubstring,
			buffer.env("EXCLUDES"),
			buffer.env("TODO"))

		buffer.out()

		buffer.out(cliInvocation)

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

		dryRunSubstring := ""

		if dryRun {
			dryRunSubstring = "--dry-run"
		}

		cliInvocation := fmt.Sprintf("tarsnap --humanize-numbers --configfile \"%s\" -c %s %s -f \"%s\" %s",
			tarsnapCredentials.ConfigFilePath,
			dryRunSubstring,
			buffer.env("EXCLUDES"),
			buffer.env("BACKUP_DATE_TIME"),
			buffer.env("TODO"))

		buffer.out()

		buffer.out(cliInvocation)

	} else if configType == model.Kopia {

		kopiaCredentials, err := config.GetKopiaCredential()
		if err != nil {
			return nil, err
		}

		// TODO: metadata name

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
		// buffer.out(fmt.Sprintf("AWS_ACCESS_KEY_ID=\"%s\"", resticCredential.S3.AccessKeyID))
		// buffer.out(fmt.Sprintf("AWS_SECRET_ACCESS_KEY=\"%s\"", resticCredential.S3.SecretAccessKey))

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

		cliInvocation = fmt.Sprintf("kopia policy set --global %s", buffer.env("EXCLUDES"))
		buffer.out(cliInvocation)

		buffer.out()
		buffer.header("Create snapshot")

		cliInvocation = fmt.Sprintf("kopia snapshot create %s", buffer.env("TODO"))
		buffer.out(cliInvocation)

		// TODO: Add tag to Kopia ?

	} else if configType == model.Robocopy {

		robocopyCredentials, err := config.GetRobocopyCredential()
		if err != nil {
			return nil, err
		}

		if robocopyCredentials.DestinationFolder == "" {
			return nil, errors.New("missing destination folder")
		}

		if robocopyCredentials.Switches == " " {
			return nil, errors.New("missing switches")
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
		buffer.out(fmt.Sprintf("set "+envName+"=\"%s\"", value))
	} else {
		buffer.out(fmt.Sprintf(envName+"=\"%s\"", value))
	}
}

func (buffer *OutputBuffer) env(envName string) string {
	if buffer.isWindows {
		return "%" + envName
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

func (buffer *OutputBuffer) comment(str string) {
	if buffer.isWindows {
		buffer.out("REM " + str)
	} else {
		buffer.out("# " + str)
	}

}

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
