package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "...",
	Long:  "...",
	Run: func(cmd *cobra.Command, args []string) {
		newAnalysis(args)
		// simpleAnalysis()
	},
}

func newAnalysis(configFiles []string) {

	isAbsolutePath := func(in string) bool {
		if runtime.GOOS == "windows" {
			indexOfColonSlash := strings.Index(in, ":\\")
			if indexOfColonSlash == 1 {
				return true
			} else {
				return false
			}
		} else {
			return strings.HasPrefix(in, "/")
		}
	}

	isExcludedPath := func(filename string, exclude string) bool {

		if isAbsolutePath(exclude) {
			reportCLIErrorAndExit(fmt.Errorf("absolute path in global exclude not supported"))
			return false
		}

		exclude, _ = strings.CutSuffix(exclude, string(os.PathSeparator))

		exclude, _ = strings.CutSuffix(exclude, "/")

		if exclude == filename {
			fmt.Println(filename, "is excluded")
			return true
		}

		return false

	}

	type expandedFolderPath struct {
		folderPath string
		excludes   []string
	}

	type NewConfigFileEntry struct {
		cfe                 ConfigFileEntry
		expandedFolderPaths []expandedFolderPath
	}

	backupFolders := map[string][]NewConfigFileEntry{}

	// var configFileStructs []NewConfigFileEntry

	for _, configFile := range configFiles {

		model, err := model.ReadConfigFile(configFile)
		if err != nil {
			reportCLIErrorAndExit(err)
			return
		}

		ncfe := NewConfigFileEntry{
			cfe: ConfigFileEntry{
				path:       configFile,
				configFile: model,
			},
		}

		for _, folder := range ncfe.cfe.configFile.Folders {

			expandedPath, err := util.Expand(folder.Path, ncfe.cfe.configFile.Substitutions)
			if err != nil {
				reportCLIErrorAndExit(err)
				return
			}
			if expandedPath == "" {
				reportCLIErrorAndExit(fmt.Errorf("expanded path is empty"))
				return
			}

			ncfe.expandedFolderPaths = append(ncfe.expandedFolderPaths, expandedFolderPath{
				folderPath: expandedPath,
				excludes:   folder.Excludes,
			})
		}

		// configFileStructs = append(configFileStructs, ncfe)

		for _, expandedFolder := range ncfe.expandedFolderPaths {

			if _, err := os.Stat(expandedFolder.folderPath); err != nil {
				reportCLIErrorAndExit(err)
				return
			}

			fileEntries, err := os.ReadDir(expandedFolder.folderPath)
			if err != nil {
				reportCLIErrorAndExit(err)
				return
			}

		outer:
			for _, fileEntry := range fileEntries {

				for _, globalExclude := range ncfe.cfe.configFile.GlobalExcludes {

					if isExcludedPath(fileEntry.Name(), globalExclude) {
						continue outer
					}
				}

				if ncfe.cfe.configFile.RobocopySettings != nil {
					reportCLIErrorAndExit(fmt.Errorf("robocopy not supported"))
					return
				}

				for _, exclude := range expandedFolder.excludes {

					if isExcludedPath(fileEntry.Name(), exclude) {
						continue outer
					}

				}

				filePath := filepath.Join(expandedFolder.folderPath, fileEntry.Name())

				allFilePaths := []string{}
				recursePath(filePath, 6, &allFilePaths)

				for _, filePathEntry := range allFilePaths {

					backupFolders[normalizePath(filePathEntry)] = append(backupFolders[normalizePath(filePathEntry)], ncfe)
				}

			}
		}
	}

	backupFolderKeys := maps.Keys(backupFolders)
	sort.Slice(backupFolderKeys, func(i, j int) bool {
		return strings.ToLower(backupFolderKeys[i]) < strings.ToLower(backupFolderKeys[j])
	})

	for _, key := range backupFolderKeys {

		v, exists := backupFolders[key]
		if !exists {
			reportCLIErrorAndExit(fmt.Errorf("key not found"))
			return
		}

		if len(v) != 1 {
			continue
		}

		fmt.Print(key + " | ")

		for _, ve := range v {
			fmt.Print(ve.cfe.path + " ")
		}
		fmt.Println()

	}

	fmt.Println("backup keys size", len(backupFolderKeys))
}

func recursePath(absPath string, maxSlashes int, res *[]string) {

	if len(strings.Split(absPath, "\\")) >= maxSlashes {
		return
	}

	if fileInfo, err := os.Stat(absPath); err != nil {
		reportCLIErrorAndExit(err)
		return
	} else if !fileInfo.IsDir() {
		fullPath := absPath
		*res = append(*res, fullPath)
		return
	}

	fileEntries, err := os.ReadDir(absPath)
	if err != nil {
		fmt.Println("Warning: Unable to read:", absPath)
		// reportCLIErrorAndExit(err)
		return
	}

	for _, fileEntry := range fileEntries {

		fullPath := filepath.Join(absPath, fileEntry.Name())

		if fileEntry.IsDir() {
			recursePath(fullPath, maxSlashes, res)
		} else {
			// Only add files to the list
			*res = append(*res, fullPath)
		}
	}

}

func normalizePath(in string) string {

	if runtime.GOOS == "windows" {

		indexOfColon := strings.Index(in, ":")
		if indexOfColon == -1 {
			return in
		}

		preColon := in[0:indexOfColon]
		postColon := in[indexOfColon+1:]

		return fmt.Sprintf("%s:%s", strings.ToLower(preColon), postColon)

	}

	return in
}

type ConfigFileEntry struct {
	path            string
	configFile      model.ConfigFile
	expandedFolders []string
}

func simpleAnalysis(configFiles []string) {

	var configFileStructs []ConfigFileEntry

	for _, configFile := range configFiles {

		model, err := model.ReadConfigFile(configFile)
		if err != nil {
			reportCLIErrorAndExit(err)
			return
		}

		cfe := ConfigFileEntry{
			path:       configFile,
			configFile: model,
		}

		for _, folder := range cfe.configFile.Folders {

			expandedPath, err := util.Expand(folder.Path, cfe.configFile.Substitutions)
			if err != nil {
				reportCLIErrorAndExit(err)
				return
			}
			if expandedPath == "" {
				reportCLIErrorAndExit(fmt.Errorf("expanded path is empty"))
				return
			}
			cfe.expandedFolders = append(cfe.expandedFolders, expandedPath)

		}

		configFileStructs = append(configFileStructs, cfe)

	}

	folderToConfigFile := map[string][]ConfigFileEntry{}

	for _, config := range configFileStructs {

		for _, expandedFolder := range config.expandedFolders {

			if _, err := os.Stat(expandedFolder); err != nil {
				reportCLIErrorAndExit(err)
				return
			}

			folderToConfigFile[normalizePath(expandedFolder)] = append(folderToConfigFile[expandedFolder], config)
		}
	}

	var mapKeys []string

	for key := range folderToConfigFile {
		mapKeys = append(mapKeys, key)
	}

	sort.Slice(mapKeys, func(i, j int) bool { return strings.ToLower(mapKeys[i]) < strings.ToLower(mapKeys[j]) })
	// sort.Strings(mapKeys)

	for _, child := range mapKeys {
		for _, parent := range mapKeys {

			if child == parent {
				continue
			}

			normalizedParent, _ := strings.CutSuffix(parent, string(filepath.Separator))
			normalizedParent = strings.ToLower(normalizedParent)
			normalizedChild := strings.ToLower(child)

			if strings.HasPrefix(normalizedChild, normalizedParent+string(filepath.Separator)) {

				fmt.Println("! parent:", parent, "child:", child)

				folderToConfigFile[child] = append(folderToConfigFile[child], folderToConfigFile[parent]...)
			}

		}
	}

	for _, key := range mapKeys {
		entries := folderToConfigFile[key]
		fmt.Println()
		fmt.Println(key + ":")

		pathMap := map[string]bool{}
		for _, entry := range entries {
			pathMap[entry.path] = true
		}

		keys := maps.Keys(pathMap)
		sort.Strings(keys)
		keys = slices.Compact(keys)

		for _, key := range keys {
			fmt.Println("-", key)
		}
	}

}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Args = func(cmd *cobra.Command, args []string) error {

		// if len(args) != 1 {
		// 	return fmt.Errorf("one argument required: (config file path)")
		// }

		return nil
	}
}
