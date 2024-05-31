package robocopy

import (
	diffgeneratedbackupscript "github.com/jgwest/backup-cli/util/cmds/diff-generated-backup-script"
)

func (RobocopyBackend) SupportsBackupShellScriptDiffCheck() bool {
	return true
}

func (RobocopyBackend) BackupShellScriptDiffCheck(configFilePath string, shellScriptPath string) error {

	config, err := extractAndValidateConfigFile(configFilePath)
	if err != nil {
		return err
	}

	generatedBackupShellScriptContents, err := generateBackupScriptFromConfigFile(configFilePath, config)
	if err != nil {
		return err
	}

	return diffgeneratedbackupscript.DiffGeneratedBackupShellScript(generatedBackupShellScriptContents, shellScriptPath)

}
