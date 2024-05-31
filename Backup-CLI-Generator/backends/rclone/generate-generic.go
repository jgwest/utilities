package rclone

import (
	"fmt"
)

func (RcloneBackend) SupportsGenerateGeneric() bool {
	return false
}

func (RcloneBackend) GenerateGeneric(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
