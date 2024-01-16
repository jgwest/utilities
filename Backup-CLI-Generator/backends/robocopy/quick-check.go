package robocopy

import (
	"fmt"
)

func (r RobocopyBackend) SupportsQuickCheck() bool {
	return false
}

func (r RobocopyBackend) QuickCheck(path string) error {
	return fmt.Errorf("unsupported")
}
