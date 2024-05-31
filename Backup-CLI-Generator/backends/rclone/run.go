package rclone

import (
	"fmt"
)

func (RcloneBackend) SupportsRun() bool {
	return false
}

func (RcloneBackend) Run(path string, args []string) error {
	return fmt.Errorf("unsupported")
}
