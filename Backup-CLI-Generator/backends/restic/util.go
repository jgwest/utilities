package restic

import (
	"errors"
	"fmt"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
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

	if configType != model.Restic {
		return model.ConfigFile{}, fmt.Errorf("configuration file does not support restic")
	}

	return config, nil
}

type directInvocation struct {
	Args                 []string
	EnvironmentVariables map[string]string
}

func generateResticDirectInvocation(config model.ConfigFile) (directInvocation, error) {

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return directInvocation{}, err
	}

	env := map[string]string{}
	{

		if resticCredential.S3 != nil {
			env["AWS_ACCESS_KEY_ID"] = resticCredential.S3.AccessKeyID
			env["AWS_SECRET_ACCESS_KEY"] = resticCredential.S3.SecretAccessKey
		}

		if len(resticCredential.Password) > 0 && len(resticCredential.PasswordFile) > 0 {
			return directInvocation{}, errors.New("both password and password file are specified")
		}

		if len(resticCredential.Password) > 0 {
			env["RESTIC_PASSWORD"] = resticCredential.Password

		} else if len(resticCredential.PasswordFile) > 0 {
			env["RESTIC_PASSWORD_FILE"] = resticCredential.PasswordFile

		} else {
			return directInvocation{}, errors.New("no restic password found")
		}

	}

	url := ""
	if resticCredential.S3 != nil {
		url = "s3:" + resticCredential.S3.URL
	} else if resticCredential.RESTEndpoint != "" {
		url = "rest:" + resticCredential.RESTEndpoint
	} else {
		return directInvocation{}, errors.New("unable to locate connection credentials")
	}

	cacertSubstring := []string{}
	if resticCredential.CACert != "" {
		expandedPath, err := util.Expand(resticCredential.CACert, config.Substitutions)
		if err != nil {
			return directInvocation{}, err
		}
		cacertSubstring = append(cacertSubstring, "--cacert", expandedPath)
	}

	execInvocation := []string{
		"restic",
		"-r",
		url,
		"--verbose",
	}

	execInvocation = append(execInvocation, cacertSubstring...)

	return directInvocation{Args: execInvocation, EnvironmentVariables: env}, nil
}
