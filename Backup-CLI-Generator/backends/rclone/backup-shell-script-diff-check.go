package rclone

import "fmt"

func (RcloneBackend) SupportsBackupShellScriptDiffCheck() bool {
	return false
}

func (RcloneBackend) BackupShellScriptDiffCheck(configFilePath string, shellScriptPath string) error {
	return fmt.Errorf("unsupported")
}
