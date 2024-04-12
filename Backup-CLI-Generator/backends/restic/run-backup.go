package restic

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"

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

	if err := processConfigRunBackup(path, config); err != nil {
		return err
	}

	return nil

}

func processConfigRunBackup(configFilePath string, config model.ConfigFile) error {

	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	if configType != model.Restic {
		return errors.New("backend only supports restic")
	}

	if err := generate.CheckMonitorFolders(configFilePath, config); err != nil {
		return err
	}

	res := runbackup.BackupRunObject{}

	isWindows := runtime.GOOS == "windows"

	if isWindows {
		cmd := exec.Command("cmd", "/c", "echo %DATE%-%TIME:~1%")
		var out strings.Builder
		cmd.Stdout = &out

		if err = cmd.Run(); err != nil {
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
		return fmt.Errorf("robocopy settings should not be present")
	}

	// Process folders
	// - Populate TODO env var
	{

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		processedFolders, err := generate.PopulateProcessedFolders(configType, config.Folders, config.Substitutions, map[string][]string{})
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

	return resticGenerateRunBackupInvocation(config, res)

}

func resticGenerateRunBackupInvocation(config model.ConfigFile, input runbackup.BackupRunObject) error {

	// TODO: Replace this with a call to util/restic-direct-invocation.go

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return err
	}

	env := map[string]string{}
	{

		if resticCredential.S3 != nil {
			env["AWS_ACCESS_KEY_ID"] = resticCredential.S3.AccessKeyID
			env["AWS_SECRET_ACCESS_KEY"] = resticCredential.S3.SecretAccessKey
		}

		if len(resticCredential.Password) > 0 && len(resticCredential.PasswordFile) > 0 {
			return errors.New("both password and password file are specified")
		}

		if len(resticCredential.Password) > 0 {
			env["RESTIC_PASSWORD"] = resticCredential.Password

		} else if len(resticCredential.PasswordFile) > 0 {
			env["RESTIC_PASSWORD_FILE"] = resticCredential.PasswordFile

		} else {
			return errors.New("no restic password found")
		}

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

	url := ""
	if resticCredential.S3 != nil {
		url = "s3:" + resticCredential.S3.URL
	} else if resticCredential.RESTEndpoint != "" {
		url = "rest:" + resticCredential.RESTEndpoint
	} else {
		return errors.New("unable to locate connection credentials")
	}

	cacertSubstring := []string{}
	if resticCredential.CACert != "" {
		expandedPath, err := util.Expand(resticCredential.CACert, config.Substitutions)
		if err != nil {
			return err
		}
		cacertSubstring = append(cacertSubstring, "--cacert", expandedPath)
	}

	excludesSubstring := []string{}
	if len(input.GlobalExcludes) > 0 {

		for _, globalExcludedFolder := range input.GlobalExcludes {
			excludesSubstring = append(excludesSubstring, "--exclude", globalExcludedFolder)
		}
	}

	execInvocation := []string{
		"restic",
		"-r",
		url,
		"--verbose",
	}

	execInvocation = append(execInvocation, tagSubstring...)
	execInvocation = append(execInvocation, cacertSubstring...)
	execInvocation = append(execInvocation, excludesSubstring...)
	execInvocation = append(execInvocation, "backup")
	execInvocation = append(execInvocation, input.Todo...)

	fmt.Println("env:", env)
	fmt.Println("exec:", execInvocation)

	return nil

}
