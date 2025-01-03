package tarsnap

import (
	"fmt"

	"github.com/jgwest/backup-cli/model"
)

func extractAndValidateConfigFile(path string) (model.ConfigFile, error) {

	config, err := model.ReadConfigFile(path)
	if err != nil {
		return model.ConfigFile{}, err
	}

	if config.RobocopySettings != nil {
		return model.ConfigFile{}, fmt.Errorf("tarsnap backend does not support robocopy settings")
	}

	configType, err := config.GetConfigType()
	if err != nil {
		return model.ConfigFile{}, err
	}

	if configType != model.Tarsnap {
		return model.ConfigFile{}, fmt.Errorf("configuration file does not support tarsnap")
	}

	return config, nil
}
