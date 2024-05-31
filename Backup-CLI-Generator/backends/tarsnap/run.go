package tarsnap

import (
	"fmt"
)

func (TarsnapBackend) SupportsRun() bool {
	return false
}

func (TarsnapBackend) Run(path string, args []string) error {
	return fmt.Errorf("unsupported")
}
