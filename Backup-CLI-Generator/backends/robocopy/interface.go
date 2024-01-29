package robocopy

import (
	"github.com/jgwest/backup-cli/model"
)

type RobocopyBackend struct{}

var _ model.Backend = RobocopyBackend{}

func (r RobocopyBackend) ConfigType() model.ConfigType {
	return model.Robocopy
}
