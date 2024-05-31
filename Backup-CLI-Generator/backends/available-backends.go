package backends

import (
	"github.com/jgwest/backup-cli/backends/kopia"
	"github.com/jgwest/backup-cli/backends/rclone"
	"github.com/jgwest/backup-cli/backends/restic"
	"github.com/jgwest/backup-cli/backends/robocopy"
	"github.com/jgwest/backup-cli/backends/tarsnap"
	"github.com/jgwest/backup-cli/model"
)

func AvailableBackends() []model.Backend {

	return []model.Backend{
		restic.ResticBackend{},
		kopia.KopiaBackend{},
		robocopy.RobocopyBackend{},
		tarsnap.TarsnapBackend{},
		rclone.RcloneBackend{},

		// add new implementations here:
		// sample.SampleBackend{},
	}

}
