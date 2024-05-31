package sample

import (
	"fmt"
)

func (SampleBackend) SupportsGenerateGeneric() bool {
	return false
}

func (SampleBackend) GenerateGeneric(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
