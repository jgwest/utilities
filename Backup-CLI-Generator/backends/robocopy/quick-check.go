package robocopy

import (
	"fmt"
)

func (RobocopyBackend) SupportsQuickCheck() bool {
	return false
}

func (RobocopyBackend) QuickCheck(path string) error {
	return fmt.Errorf("unsupported")
}
