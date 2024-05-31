package kopia

import (
	"fmt"
)

func (KopiaBackend) SupportsRun() bool {
	return false
}

func (KopiaBackend) Run(path string, args []string) error {
	return fmt.Errorf("unsupported")
}
