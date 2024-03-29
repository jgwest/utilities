package generic

import (
	"errors"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func SharedGenerateResticCredentials(config model.ConfigFile, node *util.TextNode) error {

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return err
	}
	node.Out()
	node.Header("Credentials ")

	if resticCredential.S3 != nil {
		node.SetEnv("AWS_ACCESS_KEY_ID", resticCredential.S3.AccessKeyID)
		node.SetEnv("AWS_SECRET_ACCESS_KEY", resticCredential.S3.SecretAccessKey)
	}

	if len(resticCredential.Password) > 0 && len(resticCredential.PasswordFile) > 0 {
		return errors.New("both password and password file are specified")
	}

	if len(resticCredential.Password) > 0 {
		node.SetEnv("RESTIC_PASSWORD", resticCredential.Password)

	} else if len(resticCredential.PasswordFile) > 0 {
		node.SetEnv("RESTIC_PASSWORD_FILE", resticCredential.PasswordFile)

	} else {
		return errors.New("no restic password found")
	}

	return nil

}
