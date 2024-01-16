package sample

import (
	"fmt"
)

func (r SampleBackend) SupportsGenerateBackup() bool {
	return false
}

func (r SampleBackend) GenerateBackup(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
