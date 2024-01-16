package sample

import (
	"fmt"
)

func (r SampleBackend) SupportsRun() bool {
	return false
}

func (r SampleBackend) Run(path string, args []string) error {
	return fmt.Errorf("unsupported")
}
