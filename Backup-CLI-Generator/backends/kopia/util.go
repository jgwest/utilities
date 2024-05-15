package kopia

import (
	"fmt"

	"github.com/jgwest/backup-cli/model"
)

func getAndValidateKopiaCredentials(config model.ConfigFile) (*model.KopiaCredentials, error) {
	kopiaCredentials, err := config.GetKopiaCredential()
	if err != nil {
		return nil, err
	}

	if kopiaCredentials.S3 == nil || kopiaCredentials.KopiaS3 == nil {
		return nil, fmt.Errorf("missing S3 credentials")
	}

	if kopiaCredentials.S3.AccessKeyID == "" || kopiaCredentials.S3.SecretAccessKey == "" {
		return nil, fmt.Errorf("missing S3 credential values: access key/secret access key")
	}

	if kopiaCredentials.KopiaS3.Bucket == "" || kopiaCredentials.KopiaS3.Endpoint == "" {
		return nil, fmt.Errorf("missing S3 credential values: bucket/endpoint")
	}

	if kopiaCredentials.Password == "" {
		return nil, fmt.Errorf("missing kopia password")
	}

	return &kopiaCredentials, nil
}

func extractAndValidateConfigFile(path string) (model.ConfigFile, error) {

	config, err := model.ReadConfigFile(path)
	if err != nil {
		return model.ConfigFile{}, err
	}

	if config.RobocopySettings != nil {
		return model.ConfigFile{}, fmt.Errorf("kopia backend does not support robocopy settings")
	}

	configType, err := config.GetConfigType()
	if err != nil {
		return model.ConfigFile{}, err
	}

	if configType != model.Kopia {
		return model.ConfigFile{}, fmt.Errorf("configuration file does not support kopia")
	}

	return config, nil
}
