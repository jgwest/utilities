package sample

import (
	"fmt"
)

func (SampleBackend) SupportsRun() bool {
	return false
}

func (SampleBackend) Run(path string, args []string) error {
	return fmt.Errorf("unsupported")
}
