package sample

import (
	"fmt"
)

func (SampleBackend) SupportsBackup() bool {
	return false
}

func (SampleBackend) Backup(path string, rehashSource bool) error {

	if rehashSource {
		return fmt.Errorf("unsupported flag: rehash source")
	}
	return fmt.Errorf("unsupported")
}
