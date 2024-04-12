package robocopy

import (
	"fmt"

	"github.com/jgwest/backup-cli/model"
)

func extractAndValidateConfigFile(path string) (model.ConfigFile, error) {

	config, err := model.ReadConfigFile(path)
	if err != nil {
		return model.ConfigFile{}, err
	}

	configType, err := config.GetConfigType()
	if err != nil {
		return model.ConfigFile{}, err
	}

	if configType != model.Robocopy {
		return model.ConfigFile{}, fmt.Errorf("configuration file does not support robocopy")
	}

	return config, nil
}
