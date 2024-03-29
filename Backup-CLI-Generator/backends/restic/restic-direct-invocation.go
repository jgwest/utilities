package restic

import (
	"errors"
	"fmt"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

type DirectInvocation struct {
	Args                 []string
	EnvironmentVariables map[string]string
}

func GenerateResticDirectInvocation(config model.ConfigFile) (DirectInvocation, error) {

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return DirectInvocation{}, err
	}

	env := map[string]string{}
	{

		if resticCredential.S3 != nil {
			env["AWS_ACCESS_KEY_ID"] = resticCredential.S3.AccessKeyID
			env["AWS_SECRET_ACCESS_KEY"] = resticCredential.S3.SecretAccessKey
		}

		if len(resticCredential.Password) > 0 && len(resticCredential.PasswordFile) > 0 {
			return DirectInvocation{}, errors.New("both password and password file are specified")
		}

		if len(resticCredential.Password) > 0 {
			env["RESTIC_PASSWORD"] = resticCredential.Password

		} else if len(resticCredential.PasswordFile) > 0 {
			env["RESTIC_PASSWORD_FILE"] = resticCredential.PasswordFile

		} else {
			return DirectInvocation{}, errors.New("no restic password found")
		}

	}

	url := ""
	if resticCredential.S3 != nil {
		url = "s3:" + resticCredential.S3.URL
	} else if resticCredential.RESTEndpoint != "" {
		url = "rest:" + resticCredential.RESTEndpoint
	} else {
		return DirectInvocation{}, errors.New("unable to locate connection credentials")
	}

	cacertSubstring := []string{}
	if resticCredential.CACert != "" {
		expandedPath, err := util.Expand(resticCredential.CACert, config.Substitutions)
		if err != nil {
			return DirectInvocation{}, err
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

	return DirectInvocation{Args: execInvocation, EnvironmentVariables: env}, nil
}

func (di DirectInvocation) Out() {
	fmt.Println("Environment Variables:", di.EnvironmentVariables)
	fmt.Println("Args:", di.Args)
}
