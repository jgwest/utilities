package robocopy

import (
	"errors"
	"fmt"
	"os"
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

	if err := executionBackupInvocation(config, robocopyFolders, res); err != nil {
		return err
	}

	if err := generate.CheckMonitorFolders(configFilePath, config); err != nil {
		return err
	}

	return nil
}

func executionBackupInvocation(config model.ConfigFile, robocopyFolders [][]string, input runbackup.BackupRunObject) error {

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

		cliInvocation := []string{
			"robocopy",
			folderTuple[0], // srcFolder
			folderTuple[1], // destFolder
		}
		cliInvocation = append(cliInvocation, switches...)

		fmt.Println("exec:", cliInvocation)
	}

	return nil
}
