package kopia

import (
	"fmt"
)

func (r KopiaBackend) SupportsRun() bool {
	return false
}

func (r KopiaBackend) Run(path string, args []string) error {
	return fmt.Errorf("unsupported")
}
