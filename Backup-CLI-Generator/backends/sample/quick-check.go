package sample

import (
	"fmt"
)

func (SampleBackend) SupportsQuickCheck() bool {
	return false
}

func (SampleBackend) QuickCheck(path string) error {
	return fmt.Errorf("unsupported")
}
