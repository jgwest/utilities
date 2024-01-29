package robocopy

import (
	"fmt"
)

func (r RobocopyBackend) SupportsGenerateGeneric() bool {
	return true
}

func (r RobocopyBackend) GenerateGeneric(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
