package tarsnap

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jgwest/backup-cli/generate"
	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	runbackup "github.com/jgwest/backup-cli/util/cmds/run-backup"
)

func (r TarsnapBackend) SupportsBackup() bool {
	return true
}

func (r TarsnapBackend) Backup(path string) error {

	config, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	if configType != model.Tarsnap {
		return fmt.Errorf("this configuration file does not support tarsnap")
	}

	if err := ProcessRunBackupConfig(path, config, false); err != nil {
		return err
	}

	return nil

}

func ProcessRunBackupConfig(configFilePath string, config model.ConfigFile, dryRun bool) error {

	if err := generate.CheckMonitorFolders(configFilePath, config); err != nil {
		return err
	}

	res := runbackup.BackupRunObject{}

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

	if config.RobocopySettings != nil {
		return fmt.Errorf("tarsnap backend does not support robocopy settings")
	}

	// Process folders
	// - Populate TODO env var, for everything except robocopy
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

	return tarsnapGenerateInvocation3(config, dryRun, res)

}

func tarsnapGenerateInvocation3(config model.ConfigFile, dryRun bool, input runbackup.BackupRunObject) error {

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
