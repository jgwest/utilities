package sample

import (
	"fmt"
)

func (r SampleBackend) SupportsBackup() bool {
	return false
}

func (r SampleBackend) Backup(path string) error {
	return fmt.Errorf("unsupported")
}
