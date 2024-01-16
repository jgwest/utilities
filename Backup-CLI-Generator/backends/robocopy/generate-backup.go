package robocopy

import (
	"fmt"
)

func (r RobocopyBackend) SupportsGenerateBackup() bool {
	return false
}

func (r RobocopyBackend) GenerateBackup(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
