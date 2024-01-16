package kopia

import (
	"fmt"
)

func (r KopiaBackend) SupportsGenerateGeneric() bool {
	return false
}

func (r KopiaBackend) GenerateGeneric(path string, outputPath string) error {
	return fmt.Errorf("unsupported")
}
