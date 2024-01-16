package sample

import (
	"fmt"
)

func (r SampleBackend) SupportsGenerateGeneric() bool {
	return false
}

func (r SampleBackend) GenerateGeneric(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
