package kopia

import (
	diffgeneratedbackupscript "github.com/jgwest/backup-cli/util/cmds/diff-generated-backup-script"
)

func (KopiaBackend) SupportsBackupShellScriptDiffCheck() bool {
	return true
}

func (KopiaBackend) BackupShellScriptDiffCheck(configFilePath string, shellScriptPath string) error {

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
