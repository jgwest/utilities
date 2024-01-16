package kopia

import (
	"fmt"
)

func (r KopiaBackend) SupportsQuickCheck() bool {
	return false
}

func (r KopiaBackend) QuickCheck(path string) error {
	return fmt.Errorf("unsupported")
}
