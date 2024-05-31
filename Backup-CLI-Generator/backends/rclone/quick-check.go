package rclone

import (
	"fmt"
)

func (RcloneBackend) SupportsQuickCheck() bool {
	return false
}

func (RcloneBackend) QuickCheck(path string) error {
	return fmt.Errorf("unsupported")
}
