package kopia

import (
	"fmt"
)

func (KopiaBackend) SupportsGenerateGeneric() bool {
	return false
}

func (KopiaBackend) GenerateGeneric(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
