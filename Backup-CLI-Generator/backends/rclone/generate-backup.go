package rclone

import (
	"fmt"
)

func (RcloneBackend) SupportsGenerateBackup() bool {
	return false
}

func (RcloneBackend) GenerateBackup(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
