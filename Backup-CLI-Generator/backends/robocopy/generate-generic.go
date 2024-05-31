package robocopy

import (
	"fmt"
)

func (RobocopyBackend) SupportsGenerateGeneric() bool {
	return false
}

func (RobocopyBackend) GenerateGeneric(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
