package backends

import (
	"github.com/jgwest/backup-cli/backends/kopia"
	"github.com/jgwest/backup-cli/backends/restic"
	"github.com/jgwest/backup-cli/model"
)

func AvailableBackends() []model.Backend {

	return []model.Backend{
		restic.ResticBackend{},
		kopia.KopiaBackend{},
	}

}
