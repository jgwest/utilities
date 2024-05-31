package restic

import (
	"github.com/jgwest/backup-cli/model"
)

var _ model.Backend = ResticBackend{}

type ResticBackend struct{}

func (r ResticBackend) ConfigType() model.ConfigType {
	return model.Restic
}

func generateResticBackend() model.BackendStruct {

	rb := ResticBackend{}

	res := model.BackendStruct{
		ConfigType: rb.ConfigType,
	}

	return res
}
