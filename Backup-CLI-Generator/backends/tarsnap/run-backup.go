package tarsnap

import (
	"fmt"
	"os"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds/generate"
	runbackup "github.com/jgwest/backup-cli/util/cmds/run-backup"
)

func (r TarsnapBackend) SupportsBackup() bool {
	return true
}

func (r TarsnapBackend) Backup(path string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	if err := runBackupFromConfigFile(path, config, false); err != nil {
		return err
	}

	return nil

}

func runBackupFromConfigFile(configFilePath string, config model.ConfigFile, dryRun bool) error {

	res := runbackup.BackupRunObject{}

	backupDateTime, err := runbackup.GetCurrentTimeTag()
	if err != nil {
		return err
	}
	res.BackupDateTime = backupDateTime

	if len(config.GlobalExcludes) > 0 {

		for _, exclude := range config.GlobalExcludes {

			expandedValue, err := util.Expand(exclude, config.Substitutions)
			if err != nil {
				return err
			}

			res.GlobalExcludes = append(res.GlobalExcludes, expandedValue)
		}

	}

	// Process folders
	// - Populate TODO env var
	{

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		// - This function also updates kopiaPolicyExcludes, if applicable.
		processedFolders, err := generate.PopulateProcessedFolders(model.Tarsnap, config.Folders, config.Substitutions, map[string][]string{})
		if err != nil {
			return fmt.Errorf("unable to populateProcessedFolder: %v", err)
		}

		for _, processedFolder := range processedFolders {

			folderPath, ok := (processedFolder[0]).(string)
			if !ok {
				return fmt.Errorf("invalid non-robocopy folderPath")
			}

			// The unsubstituted path is used here
			res.Todo = append(res.Todo, folderPath)

		}
	}

	if err := executeBackupInvocation(config, dryRun, res); err != nil {
		return err
	}

	if err := generate.CheckMonitorFoldersForMissingChildren(configFilePath, config); err != nil {
		return err
	}

	return nil

}

func executeBackupInvocation(config model.ConfigFile, dryRun bool, input runbackup.BackupRunObject) error {

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
		backupName += input.BackupDateTime
	}

	dryRunSubstring := []string{}
	if dryRun {
		dryRunSubstring = []string{"--dry-run"}
	}

	excludesSubstring := []string{}
	if len(input.GlobalExcludes) > 0 {
		for _, globalExcludedFolder := range input.GlobalExcludes {
			excludesSubstring = append(excludesSubstring, "--exclude", globalExcludedFolder)
		}
	}

	execInvocation := []string{
		"tarsnap",
		"--humanize-numbers",
		"--configfile",
		tarsnapCredentials.ConfigFilePath,
		"-c",
	}

	execInvocation = append(execInvocation, dryRunSubstring...)
	execInvocation = append(execInvocation, excludesSubstring...)

	execInvocation = append(execInvocation, "-f", backupName)

	execInvocation = append(execInvocation, input.Todo...)

	fmt.Println("exec:", execInvocation)

	return nil
}
