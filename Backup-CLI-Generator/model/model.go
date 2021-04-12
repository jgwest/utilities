package model

import "errors"

type ConfigFile struct {
	Substitutions  []Substitution  `yaml:"substitutions,omitempty"`
	Credentials    []Credentials   `yaml:"credentials,omitempty"`
	GlobalExcludes []string        `yaml:"globalExcludes,omitempty"`
	Folders        []Folder        `yaml:"folders,omitempty"`
	MonitorFolders []MonitorFolder `yaml:"monitorFolders,omitempty"`
}

type MonitorFolder struct {
	Path     string   `yaml:"path"`
	Excludes []string `yaml:"excludes"`
}

type Folder struct {
	Path     string   `yaml:"path"`
	Excludes []string `yaml:"excludes,omitempty"`
}

type Substitution struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Credentials struct {
	Restic  *ResticCredentials  `yaml:"restic,omitempty"`
	Kopia   *KopiaCredentials   `yaml:"kopia,omitempty"`
	Tarsnap *TarsnapCredentials `yaml:"tarsnap,omitempty"`
}

type TarsnapCredentials struct {
	ConfigFilePath string `yaml:"configFilePath"`
}

type KopiaCredentials struct {
	Password string         `yaml:"password"`
	S3       *S3Credentials `yaml:"s3"`
}

type ResticCredentials struct {
	Password     string         `yaml:"password,omitempty"`
	PasswordFile string         `yaml:"passwordFile,omitempty"`
	S3           *S3Credentials `yaml:"s3,omitempty"`
}

type S3Credentials struct {
	AccessKeyID     string `yaml:"accessKeyID"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	URL             string `yaml:"url"`
}

type ConfigType string

const (
	Restic  = "Restic"
	Kopia   = "Kopia"
	Tarsnap = "Tarsnap"
)

func (cf *ConfigFile) GetConfigType() (ConfigType, error) {

	if len(cf.Credentials) != 1 {
		return "", errors.New("unexpected number of credentials")
	}

	for _, credential := range cf.Credentials {

		count := 0
		if credential.Kopia != nil {
			count++
		}

		if credential.Restic != nil {
			count++
		}

		if credential.Tarsnap != nil {
			count++
		}

		if count != 1 {
			return "", errors.New("unexpected number of entries found in credential")
		}

	}

	credential := cf.Credentials[0]

	if credential.Kopia != nil {
		return Kopia, nil
	}

	if credential.Restic != nil {
		return Restic, nil
	}

	if credential.Tarsnap != nil {
		return Tarsnap, nil
	}

	return "", errors.New("no credentials found")
}
