package generate

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

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

	result, err := ProcessConfig(path, model)
	if err != nil {
		return err
	}

	// TODO: Add robocopy

	// TODO: Add kopia

	// TODO: Add tarsnap

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

		expandedPath := os.ExpandEnv(folder.Path)

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

func ProcessConfig(configFilePath string, config model.ConfigFile) (*OutputBuffer, error) {

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

	buffer.out()
	buffer.comment("Verify the YAML file still produces this script")
	buffer.out("backup-cli check \"" + configFilePath + "\" " + buffer.env("SCRIPTPATH"))

	// \/ \/ \/ \/

	// TODO: WARNING - THIS WILL BREAK CRONTAB BACKUPS!!!!!!!!!!!!!!!!!!!!!!

	// /\ /\ /\ /\

	// Process Global Excludes
	if len(config.GlobalExcludes) > 0 {
		buffer.out()
		buffer.comment("Excludes")
		for index, exclude := range config.GlobalExcludes {

			substring := ""

			if index > 0 {
				// substring = "$EXCLUDES "
				substring = buffer.env("EXCLUDES") + " "
			}

			buffer.setEnv("EXCLUDES", substring+"--exclude '"+exclude+"'")
			// buffer.out("EXCLUDES=\"" + substring + "--exclude '" + exclude + "'\"")

		}
	}
	configType, err := config.GetConfigType()
	if err != nil {
		return nil, err
	}

	if len(config.Substitutions) > 0 {
		return nil, errors.New("substitutions are not supported")
	}

	// Process folders
	if len(config.Folders) == 0 {
		return nil, errors.New("at least one folder is required")
	}

	buffer.out("")
	buffer.comment("Folders")

	for index, folder := range config.Folders {

		if len(folder.Excludes) != 0 && (configType == model.Restic || configType == model.Tarsnap) {
			return nil, fmt.Errorf("backup utility '%s' does not support excludes", configType)
		}

		folderPath := os.ExpandEnv(folder.Path)

		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: '%s'", folder.Path)
		}

		substring := ""

		if index > 0 {
			substring = buffer.env("TODO") + " "
			// substring = "$TODO "
		}

		// The unsubstituted path is used here
		buffer.setEnv("TODO", fmt.Sprintf("%s'%s'", substring, folder.Path))
		// buffer.out(fmt.Sprintf("TODO=\"%s'%s'\"", substring, folder.Path))
	}

	if configType == model.Restic {
		resticCredential, err := config.GetResticCredential()
		if err != nil {
			return nil, err
		}

		if resticCredential.S3 != nil {
			buffer.out()
			buffer.comment("Credentials ")
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

		cliInvocation := fmt.Sprintf("restic -r s3:%s --verbose %s backup %s", resticCredential.S3.URL, buffer.env("EXCLUDES"), buffer.env("TODO"))

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

		// TODO: tarsnap dryrun

		// tarsnap --humanize-numbers --configfile ~/tarsnap/tarsnap.conf -c (excludes)
		// 	 -f general-backup-`date +%F_%H:%M:%S` (todo)

		// TODO: general-backup

		cliInvocation := fmt.Sprintf("tarsnap --humanize-numbers --configfile \"%s\" -c %s -f ", tarsnapCredentials.ConfigFilePath, buffer.env("EXCLUDES"))

		buffer.out()

		buffer.out(cliInvocation)

	} else {
		return nil, errors.New("unsupported config")
	}

	return &buffer, nil
}

type OutputBuffer struct {
	isWindows bool
	lines     []string
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
