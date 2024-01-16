package kopia

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jgwest/backup-cli/backup"
	"github.com/jgwest/backup-cli/generate"
	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func (r KopiaBackend) SupportsBackup() bool {
	return false
}

func (r KopiaBackend) Backup(path string) error {
	config, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	if configType != model.Kopia {
		return fmt.Errorf("configuration file does not support kopia")
	}

	if err := processRunBackupConfig(path, config); err != nil {
		return err
	}

	return nil

}

func processRunBackupConfig(configFilePath string, config model.ConfigFile) error {

	if err := generate.CheckMonitorFolders(configFilePath, config); err != nil {
		return err
	}

	res := backup.BackupRunObject{}

	isWindows := runtime.GOOS == "windows"

	if isWindows {
		cmd := exec.Command("cmd", "/c", "echo %DATE%-%TIME:~1%")
		var out strings.Builder
		cmd.Stdout = &out

		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}

		res.BackupDateTime = out.String()

	} else {
		// 	backupDateTime.Out("BACKUP_DATE_TIME=`date +%F_%H:%M:%S`")

		return fmt.Errorf("linux is unsupported")
	}

	if len(config.GlobalExcludes) > 0 {

		for _, exclude := range config.GlobalExcludes {

			expandedValue, err := util.Expand(exclude, config.Substitutions)
			if err != nil {
				return err
			}

			res.GlobalExcludes = append(res.GlobalExcludes, expandedValue)
		}

	}

	// Robocopy only: Populate EXCLUDES
	if config.RobocopySettings != nil {

		return fmt.Errorf("robocopy settings found in configuration file")

	}

	// key: path to be backed up
	// value: list of excludes for that path
	kopiaPolicyExcludes := map[string][]string{}

	// Process folders
	// - Populate TODO env var, for everything except robocopy
	// - For robocopy, populate robocopyFolders
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
	return kopiaGenerateRunBackupInvocation(kopiaPolicyExcludes, config, res)

}

func kopiaGenerateRunBackupInvocation(kopiaPolicyExcludes map[string][]string, config model.ConfigFile, input backup.BackupRunObject) error {

	kopiaCredentials, err := config.GetKopiaCredential()
	if err != nil {
		return err
	}

	if kopiaCredentials.S3 == nil || kopiaCredentials.KopiaS3 == nil {
		return fmt.Errorf("missing S3 credentials: credential values")
	}

	if kopiaCredentials.S3.AccessKeyID == "" || kopiaCredentials.S3.SecretAccessKey == "" {
		return fmt.Errorf("missing S3 credential values: access key/secret access key")
	}

	if kopiaCredentials.KopiaS3.Bucket == "" || kopiaCredentials.KopiaS3.Endpoint == "" {
		return fmt.Errorf("missing S3 credential values: endpoint/bucket")
	}

	if kopiaCredentials.Password == "" {
		return fmt.Errorf("missing kopia password")
	}

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
	fmt.Println("exec:", repositoryConnectInvocation)

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

		fmt.Println("exec:", excludePolicyInvocation)
	}

	// Build
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

			fmt.Println("exec:", cliInvocation)
		}
	}

	descriptionSubstring := []string{}
	if config.Metadata != nil && config.Metadata.Name != "" {
		description := config.Metadata.Name

		if config.Metadata.AppendDateTime {
			description += input.BackupDateTime
		}

		descriptionSubstring = append(descriptionSubstring, "--description="+description)
	}

	cliInvocation := []string{
		"kopia", "snapshot", "create",
	}

	cliInvocation = append(cliInvocation, descriptionSubstring...)
	cliInvocation = append(cliInvocation, input.Todo...)

	fmt.Println("exec:", cliInvocation)

	return nil
}
