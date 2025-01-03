package tarsnap

import (
	diffgeneratedbackupscript "github.com/jgwest/backup-cli/util/cmds/diff-generated-backup-script"
)

func (TarsnapBackend) SupportsBackupShellScriptDiffCheck() bool {
	return true
}

func (TarsnapBackend) BackupShellScriptDiffCheck(configFilePath string, shellScriptPath string) error {

	config, err := extractAndValidateConfigFile(configFilePath)
	if err != nil {
		return err
	}

	generatedBackupShellScriptContents, err := generateBackupScriptFromConfigFile(configFilePath, config, false)
	if err != nil {
		return err
	}

	return diffgeneratedbackupscript.DiffGeneratedBackupShellScript(generatedBackupShellScriptContents, shellScriptPath)

}
