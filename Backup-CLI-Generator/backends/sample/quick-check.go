package sample

import (
	"fmt"
)

func (r SampleBackend) SupportsQuickCheck() bool {
	return false
}

func (r SampleBackend) QuickCheck(path string) error {
	return fmt.Errorf("unsupported")
}
