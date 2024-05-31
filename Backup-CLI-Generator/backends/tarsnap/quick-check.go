package tarsnap

import (
	"fmt"
)

func (TarsnapBackend) SupportsQuickCheck() bool {
	return false
}

func (TarsnapBackend) QuickCheck(path string) error {
	return fmt.Errorf("unsupported")
}
