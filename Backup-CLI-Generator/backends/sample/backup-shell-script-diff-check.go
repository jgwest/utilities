package sample

import "fmt"

func (r SampleBackend) SupportsBackupShellScriptDiffCheck() bool {
	return false
}

func (r SampleBackend) BackupShellScriptDiffCheck(configFilePath string, shellScriptPath string) error {
	return fmt.Errorf("unsupported")
}
