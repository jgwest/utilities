package robocopy

import (
	"fmt"
)

func (r RobocopyBackend) SupportsBackup() bool {
	return false
}

func (r RobocopyBackend) Backup(path string) error {
	return fmt.Errorf("unsupported")
}
