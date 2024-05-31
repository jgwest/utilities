package generate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

// checkMonitorFolders verifies that there are no unignored child folders of monitor folders.
func CheckMonitorFoldersForMissingChildren(configFilePath string, config model.ConfigFile) error {

	if len(config.MonitorFolders) == 0 {
		return nil
	}

	// Expand the folders to backup (ensuring they exist)
	expandedBackupPaths := []string{}
	for _, folder := range config.Folders {

		expandedPath, err := util.Expand(folder.Path, config.Substitutions)
		if err != nil {
			return err
		}

		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			return fmt.Errorf("'folders' path does not exist: '%s'", folder.Path)
		}

		expandedBackupPaths = append(expandedBackupPaths, expandedPath)
	}

	for _, monitorFolder := range config.MonitorFolders {

		monitorPath, err := util.Expand(monitorFolder.Path, config.Substitutions)
		if err != nil {
			return err
		}

		// If the paths to backup contain the monitor path itself, then we are good, so continue to the next item
		if backupPathContains(expandedBackupPaths, monitorPath) {
			continue
		}

		unbackedupPaths, err := findUnbackedUpPaths(monitorPath, monitorFolder, expandedBackupPaths)
		if err != nil {
			return err
		}

		if len(unbackedupPaths) != 0 {
			fmt.Println()
			fmt.Println("Un-backed-up paths found:")
			for _, ubPath := range unbackedupPaths {
				rel, err := filepath.Rel(monitorPath, ubPath)
				if err != nil {
					return err
				}
				fmt.Println("      - " + rel + "  # " + ubPath)
			}
			fmt.Println()

			return fmt.Errorf("monitor folder contained un-backed-up path: %v", unbackedupPaths)
		}

	}

	return nil
}

type PopulateProcessFoldersResultEntry struct {
	SrcFolderPath string
	Folder        model.Folder
}

// PopulateProcessedFolders performs error checking on config file folders, then returns
// a tuple containing (folder path to backup, folder object)
func PopulateProcessedFolders(configType model.ConfigType, configFolders []model.Folder, configFileSubstitutions []model.Substitution, kopiaPolicyExcludes map[string][]string) ([]PopulateProcessFoldersResultEntry, error) {

	var processedFolders []PopulateProcessFoldersResultEntry
	// Array of interfaces, containing:
	// - path of folder to backup
	// - the corresponding 'Folder' object

	// Populate processedFolders with list of folders to backup, and perform sanity tests
	checkDupesMap := map[string] /* source folder path -> not used */ interface{}{}
	for _, folder := range configFolders {

		if folder.Robocopy != nil && configType != model.Robocopy {
			return nil, fmt.Errorf("backup utility '%s' does not support robocopy folder entries", configType)
		}

		srcFolderPath, err := util.Expand(folder.Path, configFileSubstitutions)
		if err != nil {
			return nil, err
		}

		if _, err := os.Stat(srcFolderPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: '%s'", srcFolderPath)
		}

		if _, contains := checkDupesMap[srcFolderPath]; contains {
			return nil, fmt.Errorf("backup path list contains duplicate path: '%s'", srcFolderPath)
		}

		if len(folder.Excludes) != 0 &&
			(configType == model.Restic ||
				configType == model.Tarsnap ||
				configType == model.Robocopy) {
			return nil, fmt.Errorf("backup utility '%s' does not support local excludes", configType)

		} else if configType == model.Kopia {
			kopiaPolicyExcludes[srcFolderPath] = append(kopiaPolicyExcludes[srcFolderPath], folder.Excludes...)
		}

		processedFolders = append(processedFolders, PopulateProcessFoldersResultEntry{SrcFolderPath: srcFolderPath, Folder: folder})
	}

	return processedFolders, nil

}

func findUnbackedUpPaths(monitorPath string, monitorFolder model.MonitorFolder, expandedBackupPaths []string) ([]string, error) {

	if _, err := os.Stat(monitorPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("'monitor path' does not exist: '%s' (%s)", monitorPath, monitorFolder.Path)
	}

	pathInfo, err := os.ReadDir(monitorPath)
	if err != nil {
		return nil, err
	}

	unbackedupPaths := []string{}

	// For each child folder under the monitor folder...
outer:
	for _, monPathInfo := range pathInfo {
		if !monPathInfo.IsDir() {
			continue
		}

		fullPathName := filepath.Join(monitorPath, monPathInfo.Name())

		// For each of the excluded directories in the monitor folder, expand the glob if needed,
		// then see if the fullPathName is one of the glob matches; if so, skip it.
		for _, exclude := range monitorFolder.Excludes {

			globMatches, err := filepath.Glob(filepath.Join(monitorPath, exclude))
			if err != nil {
				return nil, err
			}

			// The folder from the folder list matched an exclude, so skip it
			if backupPathContains(globMatches, fullPathName) {
				continue outer
			}
		}

		// For each of the folders under monitorPath, ensure they are backed up
		if !backupPathContains(expandedBackupPaths, fullPathName) {
			unbackedupPaths = append(unbackedupPaths, fullPathName)
		}
	}

	return unbackedupPaths, nil

}

func backupPathContains(backupPaths []string, testStr string) bool {
	for _, backupPath := range backupPaths {
		if testStr == backupPath {
			return true
		}
	}
	return false
}
