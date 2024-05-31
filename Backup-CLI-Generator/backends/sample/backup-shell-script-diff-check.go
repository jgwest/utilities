package sample

import "fmt"

func (SampleBackend) SupportsBackupShellScriptDiffCheck() bool {
	return false
}

func (SampleBackend) BackupShellScriptDiffCheck(configFilePath string, shellScriptPath string) error {
	return fmt.Errorf("unsupported")
}
