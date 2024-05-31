package rclone

import (
	"github.com/jgwest/backup-cli/model"
)

type RcloneBackend struct{}

var _ model.Backend = RcloneBackend{}

func (RcloneBackend) ConfigType() model.ConfigType {
	return model.Rclone
}
