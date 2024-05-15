package kopia

import (
	"fmt"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds/generate"
	runbackup "github.com/jgwest/backup-cli/util/cmds/run-backup"
)

func (r KopiaBackend) SupportsBackup() bool {
	return true
}

func (r KopiaBackend) Backup(path string) error {

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

	// key: path to be backed up
	// value: list of excludes for that path
	kopiaPolicyExcludes := map[string][]string{}

	// Process folders
	// - Populate TODO list
	{

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		// - This function also updates kopiaPolicyExcludes, if applicable.
		processedFolders, err := generate.PopulateProcessedFolders(model.Kopia, config.Folders, config.Substitutions, kopiaPolicyExcludes)
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

	// Uses TODO, BACKUP_DATE_TIME, EXCLUDES, from above
	if err := executeBackupInvocation(kopiaPolicyExcludes, config, res); err != nil {
		return err
	}

	if err := generate.CheckMonitorFoldersForMissingChildren(configFilePath, config); err != nil {
		return err
	}

	return nil
}

func executeBackupInvocation(kopiaPolicyExcludes map[string][]string, config model.ConfigFile, input runbackup.BackupRunObject) error {

	kopiaCredentials, err := getAndValidateKopiaCredentials(config)
	if err != nil {
		return err
	}

	// Connect the repository
	{
		repositoryConnectInvocation := []string{
			"kopia",
			"repository",
			"connect",
			"s3",
			"--bucket=" + kopiaCredentials.KopiaS3.Bucket,
			"--access-key=" + kopiaCredentials.S3.AccessKeyID,
			"--secret-access-key=" + kopiaCredentials.S3.SecretAccessKey,
			"--password=" + kopiaCredentials.Password,
			"--endpoint=" + kopiaCredentials.KopiaS3.Endpoint,
			"--region=" + kopiaCredentials.KopiaS3.Region,
		}

		repositoryConnectDI := util.DirectInvocation{
			Args:                 repositoryConnectInvocation,
			EnvironmentVariables: map[string]string{},
		}

		if err := repositoryConnectDI.Execute(); err != nil {
			return err
		}
	}

	// Set the global policy
	if len(input.GlobalExcludes) > 0 {
		excludePolicyInvocation := []string{
			"kopia",
			"policy",
			"set",
			"--global",
		}

		for _, globalExcludedFolder := range input.GlobalExcludes {
			excludePolicyInvocation = append(excludePolicyInvocation, "--add-ignore", globalExcludedFolder)
		}

		setPolicyDI := util.DirectInvocation{
			Args:                 excludePolicyInvocation,
			EnvironmentVariables: map[string]string{},
		}

		if err := setPolicyDI.Execute(); err != nil {
			return err
		}

	}

	// Run set policy for the local paths
	if len(kopiaPolicyExcludes) > 0 {

		for backupPath, excludes := range kopiaPolicyExcludes {

			if len(excludes) == 0 {
				continue
			}

			excludesStr := []string{}
			for _, exclude := range excludes {
				excludesStr = append(excludesStr, "--add-ignore", exclude)
			}

			cliInvocation := []string{
				"kopia", "policy", "set",
			}
			cliInvocation = append(cliInvocation, excludesStr...)

			cliInvocation = append(cliInvocation, backupPath)

			localPolicyDI := util.DirectInvocation{
				Args:                 cliInvocation,
				EnvironmentVariables: map[string]string{},
			}

			if err := localPolicyDI.Execute(); err != nil {
				return err
			}

		}
	}

	// Finally, create a snapshot

	descriptionSubstring := []string{}
	if config.Metadata != nil && config.Metadata.Name != "" {
		description := config.Metadata.Name

		if config.Metadata.AppendDateTime {
			description += input.BackupDateTime
		}

		descriptionSubstring = append(descriptionSubstring, "--description="+description)
	}

	createsSnaphotDI := []string{
		"kopia", "snapshot", "create",
	}

	createsSnaphotDI = append(createsSnaphotDI, descriptionSubstring...)
	createsSnaphotDI = append(createsSnaphotDI, input.Todo...)

	directionInvocation := util.DirectInvocation{
		Args:                 createsSnaphotDI,
		EnvironmentVariables: map[string]string{},
	}

	return directionInvocation.Execute()
}
