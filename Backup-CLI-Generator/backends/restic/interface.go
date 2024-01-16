package restic

import (
	"github.com/jgwest/backup-cli/model"
)

var _ model.Backend = ResticBackend{}

type ResticBackend struct{}

func (r ResticBackend) ConfigType() model.ConfigType {
	return model.Restic
}
