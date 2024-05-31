package kopia

import (
	"fmt"
)

func (KopiaBackend) SupportsQuickCheck() bool {
	return false
}

func (KopiaBackend) QuickCheck(path string) error {
	return fmt.Errorf("unsupported")
}
