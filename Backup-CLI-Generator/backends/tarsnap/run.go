package tarsnap

import (
	"fmt"
)

func (r TarsnapBackend) SupportsRun() bool {
	return false
}

func (r TarsnapBackend) Run(path string, args []string) error {
	return fmt.Errorf("unsupported")
}
