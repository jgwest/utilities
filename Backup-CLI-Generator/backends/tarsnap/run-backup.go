package tarsnap

import (
	"fmt"
)

func (r TarsnapBackend) SupportsBackup() bool {
	return false
}

func (r TarsnapBackend) Backup(path string) error {
	return fmt.Errorf("unsupported")
}
