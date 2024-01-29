package tarsnap

import (
	"github.com/jgwest/backup-cli/model"
)

type TarsnapBackend struct{}

var _ model.Backend = TarsnapBackend{}

func (r TarsnapBackend) ConfigType() model.ConfigType {
	return model.Tarsnap
}
