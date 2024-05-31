package rclone

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/jgwest/backup-cli/util/cmds/generate"
	runbackup "github.com/jgwest/backup-cli/util/cmds/run-backup"
)

func (RcloneBackend) SupportsBackup() bool {
	return true
}

func (RcloneBackend) Backup(path string, rehashSource bool) error {

	if rehashSource {
		return fmt.Errorf("unsupported flag: rehash source")
	}

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	if err := runBackupFromConfigFile(path, config); err != nil {
		return err
	}

	return nil

}

type sourceToDestFolder struct {
	// source is folder path
	source string
	// dest is destination folder with basename of source folder appended
	dest string
}

func runBackupFromConfigFile(configFilePath string, config model.ConfigFile) error {

	res := runbackup.BackupRunObject{}

	if len(config.GlobalExcludes) > 0 {

		for _, exclude := range config.GlobalExcludes {

			expandedValue, err := util.Expand(exclude, config.Substitutions)
			if err != nil {
				return err
			}

			res.GlobalExcludes = append(res.GlobalExcludes, expandedValue)
		}

	}

	// rcloneFolders contains a slice of:
	// - source folder path
	// - destination folder (with basename of source folder appended)
	// Example:
	// - [C:\Users] -> [B:\backup\C-Users]
	// - [D:\Users] -> [B:\backup\D-Users]
	// - [C:\To-Backup] -> [B:\backup\To-Backup]
	var rcloneFolders []sourceToDestFolder

	// Process folders
	// - Populate TODO env var, for everything except robocopy
	// - For robocopy, populate rcloneFolders
	{

		// processFolder is a slice of: [string (path to backup), model.Folder (folder object)]
		// - This function also updates kopiaPolicyExcludes, if applicable.
		processedFolders, err := generate.PopulateProcessedFolders(model.Rclone, config.Folders, config.Substitutions, map[string][]string{})
		if err != nil {
			return fmt.Errorf("unable to populateProcessedFolder: %v", err)
		}

		// Ensure that none of the folders share a basename
		if err := rcloneValidateBasenames(processedFolders); err != nil {
			return err
		}

		if rcloneCredentials, err := config.GetRcloneCredential(); err == nil {

			rcloneFolders, err = rcloneGenerateTargetPaths(processedFolders, rcloneCredentials)
			if err != nil {
				return err
			}

		} else {
			return err
		}

	}

	if err := executeBackupInvocation(config, rcloneFolders, res); err != nil {
		return err
	}

	if err := generate.CheckMonitorFoldersForMissingChildren(configFilePath, config); err != nil {
		return err
	}

	return nil
}

func executeBackupInvocation(config model.ConfigFile, rcloneFolders []sourceToDestFolder, input runbackup.BackupRunObject) error {

	// rcloneCredentials, err := getAndValidateRcloneCredentials(config)
	// if err != nil {
	// 	return err
	// }
	switches := []string{}

	for _, folderTuple := range rcloneFolders {

		srcFolder, destFolder := folderTuple.source, folderTuple.dest

		cliInvocation := []string{
			"rclone",
			"sync",
			srcFolder,
			destFolder,
			"--progress",
			// "--dry-run",
			"--create-empty-src-dirs",
			"--ignore-errors",
			// "--max-delete", "1000",
			"--transfers", "8",
			"--delete-excluded",
		}
		cliInvocation = append(cliInvocation, switches...)

		for _, globalExclude := range input.GlobalExcludes {
			cliInvocation = append(cliInvocation, "--exclude", globalExclude)
		}

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

// rcloneGenerateTargetPaths returns a slice of:
// - source folder path
// - destination folder (with basename of source folder appended)
// Example:
// - [C:\Users] -> [B:\backup\C-Users]
// - [D:\Users] -> [B:\backup\D-Users]
// - [C:\To-Backup] -> [B:\backup\To-Backup]
func rcloneGenerateTargetPaths(processedFolders []generate.PopulateProcessFoldersResultEntry, rcloneCredentials model.RcloneCredentials) ([]sourceToDestFolder, error) {
	res := []sourceToDestFolder{}

	targetFolder := rcloneCredentials.DestinationFolder

	for _, rcloneFolder := range processedFolders {

		rcloneSrcFolderPath := rcloneFolder.SrcFolderPath
		folderEntry := rcloneFolder.Folder

		// Use the name of the src folder as the dest folder name, unless
		// a replacement is specified in the folder entry.
		destFolderName := filepath.Base(rcloneSrcFolderPath)
		if folderEntry.Rclone != nil && folderEntry.Rclone.DestFolderName != "" {
			destFolderName = folderEntry.Rclone.DestFolderName
		}

		// tuple:
		// - source folder path
		// - destination folder with basename of source folder appended

		tuple := sourceToDestFolder{
			source: rcloneSrcFolderPath,
			dest:   filepath.Join(targetFolder, destFolderName),
		}
		res = append(res, tuple)

	}

	return res, nil

}

// rcloneValidateBasenames ensures that none of the folders share a basename
func rcloneValidateBasenames(processedFolders []generate.PopulateProcessFoldersResultEntry) error {

	basenameMap := map[string]interface{}{}
	for _, rcloneFolder := range processedFolders {

		rcloneFolderPath := rcloneFolder.SrcFolderPath
		folderEntry := rcloneFolder.Folder

		// Use the name of the src folder as the dest folder name, unless
		// a replacement is specified in the folder entry.
		destFolderName := filepath.Base(rcloneFolderPath)
		if folderEntry.Rclone != nil && folderEntry.Rclone.DestFolderName != "" {
			destFolderName = folderEntry.Rclone.DestFolderName
		}

		if _, contains := basenameMap[destFolderName]; contains {
			return fmt.Errorf("multiple folders share the same base name: %s", destFolderName)
		}

		basenameMap[destFolderName] = destFolderName
	}
	return nil
}

func extractAndValidateConfigFile(path string) (model.ConfigFile, error) {

	config, err := model.ReadConfigFile(path)
	if err != nil {
		return model.ConfigFile{}, err
	}

	configType, err := config.GetConfigType()
	if err != nil {
		return model.ConfigFile{}, err
	}

	if configType != model.Rclone {
		return model.ConfigFile{}, fmt.Errorf("configuration file does not support rclone")
	}

	return config, nil
}

func getAndValidateRcloneCredentials(config model.ConfigFile) (*model.RcloneCredentials, error) {
	rcloneCredentials, err := config.GetRcloneCredential()
	if err != nil {
		return nil, err
	}

	if rcloneCredentials.DestinationFolder == "" {
		return nil, errors.New("missing destination folder")
	}

	if config.Metadata != nil && (config.Metadata.Name != "" || config.Metadata.AppendDateTime) {
		return nil, fmt.Errorf("metadata features are not supported with rclone")
	}

	if _, err := os.Stat(rcloneCredentials.DestinationFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("rclone destination folder does not exist: '%s'", rcloneCredentials.DestinationFolder)
	}

	return &rcloneCredentials, nil

}
