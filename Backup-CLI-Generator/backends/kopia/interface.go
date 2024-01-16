package kopia

import (
	"github.com/jgwest/backup-cli/model"
)

type KopiaBackend struct{}

var _ model.Backend = KopiaBackend{}

func (r KopiaBackend) ConfigType() model.ConfigType {
	return "sample"
}
