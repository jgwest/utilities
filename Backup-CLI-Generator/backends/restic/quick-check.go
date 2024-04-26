package restic

import (
	"github.com/jgwest/backup-cli/model"
)

func (r ResticBackend) SupportsQuickCheck() bool {
	return true
}

func (r ResticBackend) QuickCheck(path string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	return executeQuickCheck(config)

}

func executeQuickCheck(config model.ConfigFile) error {

	invocParams, err := generateResticDirectInvocation(config)
	if err != nil {
		return err
	}

	invocParams.Args = append(invocParams.Args, []string{
		"check",
	}...)

	return invocParams.Execute()
}
