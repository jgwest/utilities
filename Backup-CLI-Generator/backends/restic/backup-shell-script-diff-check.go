package restic

import (
	diffgeneratedbackupscript "github.com/jgwest/backup-cli/util/cmds/diff-generated-backup-script"
)

func (r ResticBackend) SupportsBackupShellScriptDiffCheck() bool {
	return true
}

func (r ResticBackend) BackupShellScriptDiffCheck(configFilePath string, shellScriptPath string) error {

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
