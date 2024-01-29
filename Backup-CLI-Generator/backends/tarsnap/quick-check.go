package tarsnap

import (
	"fmt"
)

func (r TarsnapBackend) SupportsQuickCheck() bool {
	return false
}

func (r TarsnapBackend) QuickCheck(path string) error {
	return fmt.Errorf("unsupported")
}
