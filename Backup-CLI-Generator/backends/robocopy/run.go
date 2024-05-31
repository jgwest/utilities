package robocopy

import (
	"fmt"
)

func (RobocopyBackend) SupportsRun() bool {
	return false
}

func (RobocopyBackend) Run(path string, args []string) error {
	return fmt.Errorf("unsupported")
}
