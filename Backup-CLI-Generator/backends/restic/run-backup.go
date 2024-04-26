package restic

import (
	"errors"
	"fmt"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds/generate"
	runbackup "github.com/jgwest/backup-cli/util/cmds/run-backup"
)

func (r ResticBackend) SupportsBackup() bool {
	return true
}

func (r ResticBackend) Backup(path string) error {

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
	// - Populate TODO list
	{

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		processedFolders, err := generate.PopulateProcessedFolders(model.Restic, config.Folders, config.Substitutions, map[string][]string{})
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

	if err := executeBackupInvocation(config, res); err != nil {
		return err
	}

	if err := generate.CheckMonitorFolders(configFilePath, config); err != nil {
		return err
	}

	return nil

}

func executeBackupInvocation(config model.ConfigFile, input runbackup.BackupRunObject) error {

	directInvocation, err := generateResticDirectInvocation(config)
	if err != nil {
		return err
	}

	tagSubstring := []string{}
	if config.Metadata != nil {
		if len(config.Metadata.Name) == 0 {
			return errors.New("metadata exists, but name is nil")
		}

		tagName := config.Metadata.Name
		if config.Metadata.AppendDateTime {
			tagName += input.BackupDateTime
		}

		tagSubstring = append(tagSubstring, "--tag", tagName)
	}

	excludesSubstring := []string{}
	if len(input.GlobalExcludes) > 0 {

		for _, globalExcludedFolder := range input.GlobalExcludes {
			excludesSubstring = append(excludesSubstring, "--exclude", globalExcludedFolder)
		}
	}

	directInvocation.Args = append(directInvocation.Args, excludesSubstring...)
	directInvocation.Args = append(directInvocation.Args, tagSubstring...)

	directInvocation.Args = append(directInvocation.Args, "backup")
	directInvocation.Args = append(directInvocation.Args, input.Todo...)

	return directInvocation.Execute()

}
