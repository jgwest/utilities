package sample

import (
	"github.com/jgwest/backup-cli/model"
)

type SampleBackend struct{}

var _ model.Backend = SampleBackend{}

func (SampleBackend) ConfigType() model.ConfigType {
	return "sample"
}
