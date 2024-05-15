package robocopy

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds/generate"
	runbackup "github.com/jgwest/backup-cli/util/cmds/run-backup"
)

func (r RobocopyBackend) SupportsBackup() bool {
	return true
}

func (r RobocopyBackend) Backup(path string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	if err := runBackupFromConfigFile(path, config); err != nil {
		return err
	}

	return nil

}

func runBackupFromConfigFile(configFilePath string, config model.ConfigFile) error {

	res := runbackup.BackupRunObject{}

	isWindows := runtime.GOOS == "windows"

	backupDateTime, err := runbackup.GetCurrentTimeTag()
	if err != nil {
		return err
	}
	res.BackupDateTime = backupDateTime

	if len(config.GlobalExcludes) > 0 {
		return errors.New("robocopy does not support global excludes")
	}

	// Robocopy only: Populate EXCLUDES
	if config.RobocopySettings != nil {

		if !isWindows {
			return errors.New("robocopy settings not supported for non-Windows")
		}

		for _, excludeFile := range config.RobocopySettings.ExcludeFiles {

			expandedValue, err := util.Expand(excludeFile, config.Substitutions)
			if err != nil {
				return err
			}

			res.RobocopyFileExcludes = append(res.RobocopyFileExcludes, expandedValue)

		}

		for _, excludeDir := range config.RobocopySettings.ExcludeFolders {

			expandedValue, err := util.Expand(excludeDir, config.Substitutions)
			if err != nil {
				return err
			}

			if strings.Contains(expandedValue, "*") {
				return fmt.Errorf("wildcards may not be supported in directories with robocopy: %s", expandedValue)
			}

			res.RobocopyFolderExcludes = append(res.RobocopyFolderExcludes, expandedValue)
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
	// - Populate TODO env var, for everything except robocopy
	// - For robocopy, populate robocopyFolders
	{

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		// - This function also updates kopiaPolicyExcludes, if applicable.
		processedFolders, err := generate.PopulateProcessedFolders(model.Robocopy, config.Folders, config.Substitutions, map[string][]string{})
		if err != nil {
			return fmt.Errorf("unable to populateProcessedFolder: %v", err)
		}

		// Ensure that none of the folders share a basename
		if err := robocopyValidateBasenames(processedFolders); err != nil {
			return err
		}

		if robocopyCredentials, err := config.GetRobocopyCredential(); err == nil {

			robocopyFolders, err = robocopyGenerateTargetPaths(processedFolders, robocopyCredentials)
			if err != nil {
				return err
			}

		} else {
			return err
		}

	}

	if err := executeBackupInvocation(config, robocopyFolders, res); err != nil {
		return err
	}

	if err := generate.CheckMonitorFoldersForMissingChildren(configFilePath, config); err != nil {
		return err
	}

	return nil
}

func executeBackupInvocation(config model.ConfigFile, robocopyFolders [][]string, input runbackup.BackupRunObject) error {

	robocopyCredentials, err := getAndValidateRobocopyCredentials(config)
	if err != nil {
		return err
	}
	switches := []string{}

	// Add switches from config gile
	switches = append(switches, strings.Fields(robocopyCredentials.Switches)...)

	// Add file and folder excludes
	for _, file := range input.RobocopyFileExcludes {
		switches = append(switches, "/XF", file)
	}

	for _, folder := range input.RobocopyFolderExcludes {
		switches = append(switches, "/XD", folder)
	}

	for _, folderTuple := range robocopyFolders {

		srcFolder, destFolder := folderTuple[0], folderTuple[1]

		cliInvocation := []string{
			"robocopy",
			srcFolder,
			destFolder,
		}
		cliInvocation = append(cliInvocation, switches...)

		robocopyDI := util.DirectInvocation{
			Args:                 cliInvocation,
			EnvironmentVariables: map[string]string{},
		}

		if err := robocopyDI.Execute(); err != nil {
			return err
		}

	}

	return nil
}
