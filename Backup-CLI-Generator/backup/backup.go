package backup

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jgwest/backup-cli/generate"
	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func RunBackup(path string) error {

	model, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	if err := ProcessConfig(path, model, false); err != nil {
		return err
	}

	return nil

}

func ProcessConfig(configFilePath string, config model.ConfigFile, dryRun bool) error {

	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	if dryRun && configType != model.Tarsnap {
		return fmt.Errorf("dryrun is only supported for tarsnap")
	}

	if err := generate.CheckMonitorFolders(configFilePath, config); err != nil {
		return err
	}

	res := BackupRunObject{}

	isWindows := runtime.GOOS == "windows"

	if isWindows {
		cmd := exec.Command("cmd", "/c", "echo %DATE%-%TIME:~1%")
		var out strings.Builder
		cmd.Stdout = &out

		if err = cmd.Run(); err != nil {
			log.Fatal(err)
		}

		res.backupDateTime = out.String()

	} else {
		// 	backupDateTime.Out("BACKUP_DATE_TIME=`date +%F_%H:%M:%S`")

		return fmt.Errorf("linux is unsupported")
	}

	if len(config.GlobalExcludes) > 0 {

		if configType == model.Robocopy {
			return errors.New("robocopy does not support global excludes")
		}

		for _, exclude := range config.GlobalExcludes {

			expandedValue, err := util.Expand(exclude, config.Substitutions)
			if err != nil {
				return err
			}

			res.globalExcludes = append(res.globalExcludes, expandedValue)
		}

	}

	// Robocopy only: Populate EXCLUDES
	if config.RobocopySettings != nil {

		if configType != model.Robocopy || !isWindows {
			return errors.New("robocopy settings not supported for non-robocopy")
		}

		for _, excludeFile := range config.RobocopySettings.ExcludeFiles {

			expandedValue, err := util.Expand(excludeFile, config.Substitutions)
			if err != nil {
				return err
			}

			res.robocopyFileExcludes = append(res.robocopyFileExcludes, expandedValue)

		}

		for _, excludeDir := range config.RobocopySettings.ExcludeFolders {

			expandedValue, err := util.Expand(excludeDir, config.Substitutions)
			if err != nil {
				return err
			}

			if strings.Contains(expandedValue, "*") {
				return fmt.Errorf("wildcards may not be supported in directories with robocopy: %s", expandedValue)
			}

			res.robocopyFolderExcludes = append(res.robocopyFolderExcludes, expandedValue)
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

	// key: path to be backed up
	// value: list of excludes for that path
	kopiaPolicyExcludes := map[string][]string{}

	// Process folders
	// - Populate TODO env var, for everything except robocopy
	// - For robocopy, populate robocopyFolders
	{

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		// - This function also updates kopiaPolicyExcludes, if applicable.
		processedFolders, err := generate.PopulateProcessedFolders(configType, config.Folders, config.Substitutions, kopiaPolicyExcludes)
		if err != nil {
			return fmt.Errorf("unable to populateProcessedFolder: %v", err)
		}

		// Everything except robocopy
		if configType == model.Kopia || configType == model.Restic || configType == model.Tarsnap {
			for _, processedFolder := range processedFolders {

				folderPath, ok := (processedFolder[0]).(string)
				if !ok {
					return fmt.Errorf("invalid non-robocopy folderPath")
				}

				// The unsubstituted path is used here
				res.todo = append(res.todo, folderPath)

			}
		} else if configType == model.Robocopy {

			// Ensure that none of the folders share a basename
			if err := generate.RobocopyValidateBasenames(processedFolders); err != nil {
				return err
			}

			if robocopyCredentials, err := config.GetRobocopyCredential(); err == nil {

				robocopyFolders, err = generate.RobocopyGenerateTargetPaths(processedFolders, robocopyCredentials)
				if err != nil {
					return err
				}

			} else {
				return err
			}

		} else { // end robocopy section
			return errors.New("unrecognized config type")
		}

	}

	if configType == model.Restic {

		return resticGenerateInvocation3(config, res)

	} else if configType == model.Tarsnap {

		return tarsnapGenerateInvocation3(config, dryRun, res)

	} else if configType == model.Kopia {

		// Uses TODO, BACKUP_DATE_TIME, EXCLUDES, from above
		return kopiaGenerateInvocation3(kopiaPolicyExcludes, config, res)

	} else if configType == model.Robocopy {

		return robocopyGenerateInvocation3(config, robocopyFolders, res)

	} else {
		return errors.New("unsupported config")
	}

}

func robocopyGenerateInvocation3(config model.ConfigFile, robocopyFolders [][]string, input BackupRunObject) error {

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
	for _, file := range input.robocopyFileExcludes {
		switches = append(switches, "/XF", file)
	}

	for _, folder := range input.robocopyFolderExcludes {
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

func kopiaGenerateInvocation3(kopiaPolicyExcludes map[string][]string, config model.ConfigFile, input BackupRunObject) error {

	kopiaCredentials, err := config.GetKopiaCredential()
	if err != nil {
		return err
	}

	if kopiaCredentials.S3 == nil || kopiaCredentials.KopiaS3 == nil {
		return fmt.Errorf("missing S3 credentials")
	}

	if kopiaCredentials.S3.AccessKeyID == "" || kopiaCredentials.S3.SecretAccessKey == "" {
		return fmt.Errorf("missing S3 credential values")
	}

	if kopiaCredentials.KopiaS3.Bucket == "" || kopiaCredentials.KopiaS3.Region == "" || kopiaCredentials.KopiaS3.Endpoint == "" {
		return fmt.Errorf("missing S3 credential values")
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

	if len(input.globalExcludes) > 0 {
		excludePolicyInvocation := []string{
			"kopia",
			"policy",
			"set",
			"--global",
		}

		for _, globalExcludedFolder := range input.globalExcludes {
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
			description += input.backupDateTime
		}

		descriptionSubstring = append(descriptionSubstring, "--description="+description)
	}

	cliInvocation := []string{
		"kopia", "snapshot", "create",
	}

	cliInvocation = append(cliInvocation, descriptionSubstring...)
	cliInvocation = append(cliInvocation, input.todo...)

	fmt.Println("exec:", cliInvocation)

	return nil
}

func tarsnapGenerateInvocation3(config model.ConfigFile, dryRun bool, input BackupRunObject) error {

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
		backupName += input.backupDateTime
	}

	dryRunSubstring := []string{}
	if dryRun {
		dryRunSubstring = []string{"--dry-run"}
	}

	excludesSubstring := []string{}
	if len(input.globalExcludes) > 0 {
		for _, globalExcludedFolder := range input.globalExcludes {
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

	execInvocation = append(execInvocation, input.todo...)

	fmt.Println("exec:", execInvocation)

	return nil
}

func resticGenerateInvocation3(config model.ConfigFile, input BackupRunObject) error {

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
			tagName += input.backupDateTime
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
	if len(input.globalExcludes) > 0 {

		for _, globalExcludedFolder := range input.globalExcludes {
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
	execInvocation = append(execInvocation, input.todo...)

	fmt.Println("env:", env)
	fmt.Println("exec:", execInvocation)

	return nil

}

type BackupRunObject struct {
	backupDateTime string

	globalExcludes []string

	robocopyFileExcludes   []string
	robocopyFolderExcludes []string

	todo []string
}
