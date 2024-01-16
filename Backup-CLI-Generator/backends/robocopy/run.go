package robocopy

import (
	"fmt"
)

func (r RobocopyBackend) SupportsRun() bool {
	return false
}

func (r RobocopyBackend) Run(path string, args []string) error {
	return fmt.Errorf("unsupported")
}
