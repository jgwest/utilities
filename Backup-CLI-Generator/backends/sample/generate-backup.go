package sample

import (
	"fmt"
)

func (SampleBackend) SupportsGenerateBackup() bool {
	return false
}

func (SampleBackend) GenerateBackup(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
