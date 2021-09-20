package model

import (
	"errors"
	"fmt"
)

type ConfigFile struct {
	Metadata         *Metadata         `yaml:"metadata,omitempty"`
	Substitutions    []Substitution    `yaml:"substitutions,omitempty"`
	Credentials      []Credentials     `yaml:"credentials,omitempty"`
	GlobalExcludes   []string          `yaml:"globalExcludes,omitempty"`
	Folders          []Folder          `yaml:"folders,omitempty"`
	MonitorFolders   []MonitorFolder   `yaml:"monitorFolders,omitempty"`
	RobocopySettings *RobocopySettings `yaml:"robocopySettings,omitempty"`
}

type Metadata struct {
	Name           string `yaml:"name"`
	AppendDateTime bool   `yaml:"appendDateTime"`
}

type MonitorFolder struct {
	Path     string   `yaml:"path"`
	Excludes []string `yaml:"excludes,omitempty"`
}

type Folder struct {
	Path     string                  `yaml:"path"`
	Excludes []string                `yaml:"excludes,omitempty"`
	Robocopy *RobocopyFolderSettings `yaml:"robocopy,omitempty"`
}

type RobocopyFolderSettings struct {
	DestFolderName string `yaml:"destFolderName"`
}

type Substitution struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Credentials struct {
	Restic   *ResticCredentials   `yaml:"restic,omitempty"`
	Kopia    *KopiaCredentials    `yaml:"kopia,omitempty"`
	Tarsnap  *TarsnapCredentials  `yaml:"tarsnap,omitempty"`
	Robocopy *RobocopyCredentials `yaml:"robocopy,omitempty"`
}

type RobocopyCredentials struct {
	Switches          string `yaml:"switches"`
	DestinationFolder string `yaml:"destinationFolder"`
}

type TarsnapCredentials struct {
	ConfigFilePath string `yaml:"configFilePath"`
}

type KopiaCredentials struct {
	Password string              `yaml:"password"`
	S3       *S3Credentials      `yaml:"s3"`
	KopiaS3  *KopiaS3Credentials `yaml:"kopiaS3"`
}

type ResticCredentials struct {
	CACert       string         `yaml:"caCert,omitempty"`
	Password     string         `yaml:"password,omitempty"`
	PasswordFile string         `yaml:"passwordFile,omitempty"`
	RESTEndpoint string         `yaml:"restEndpoint,omitempty"`
	S3           *S3Credentials `yaml:"s3,omitempty"`
}

type S3Credentials struct {
	AccessKeyID     string `yaml:"accessKeyID"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	URL             string `yaml:"url"`
}

type KopiaS3Credentials struct {
	Region   string `yaml:"region"`
	Bucket   string `yaml:"bucket"`
	Endpoint string `yaml:"endpoint"`
}

type RobocopySettings struct {
	ExcludeFiles   []string `yaml:"excludeFiles,omitempty"`
	ExcludeFolders []string `yaml:"excludeFolders,omitempty"`
}

type ConfigType string

const (
	Restic   = "Restic"
	Kopia    = "Kopia"
	Tarsnap  = "Tarsnap"
	Robocopy = "Robocopy"
)

func (cf *ConfigFile) GetConfigType() (ConfigType, error) {

	if len(cf.Credentials) != 1 {
		return "", fmt.Errorf("unexpected number of credentials: %v", len(cf.Credentials))
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

		if credential.Robocopy != nil {
			count++
		}

		if count != 1 {
			return "", fmt.Errorf("unexpected number of credentials: %v", count)
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

	if credential.Robocopy != nil {
		return Robocopy, nil
	}

	return "", errors.New("no credentials found")
}

func (cf *ConfigFile) GetRobocopyCredential() (RobocopyCredentials, error) {

	// Must have a single kopia credential
	if confType, err := cf.GetConfigType(); confType != Robocopy || err != nil {
		if err == nil {
			err = errors.New("invalid kopia credentials")
		}
		return RobocopyCredentials{}, err
	}

	return *cf.Credentials[0].Robocopy, nil
}

func (cf *ConfigFile) GetKopiaCredential() (KopiaCredentials, error) {

	// Must have a single kopia credential
	if confType, err := cf.GetConfigType(); confType != Kopia || err != nil {
		if err == nil {
			err = errors.New("invalid kopia credentials")
		}
		return KopiaCredentials{}, err
	}

	return *cf.Credentials[0].Kopia, nil
}

func (cf *ConfigFile) GetResticCredential() (ResticCredentials, error) {

	// Must have a single restic credential
	if confType, err := cf.GetConfigType(); confType != Restic || err != nil {
		if err == nil {
			err = errors.New("invalid restic credentials")
		}
		return ResticCredentials{}, err
	}

	return *cf.Credentials[0].Restic, nil
}

func (cf *ConfigFile) GetTarsnapCredential() (TarsnapCredentials, error) {

	// Must have a single tarsnap credential
	if confType, err := cf.GetConfigType(); confType != Tarsnap || err != nil {
		if err == nil {
			err = errors.New("invalid tarsnap credentials")
		}
		return TarsnapCredentials{}, err
	}

	return *cf.Credentials[0].Tarsnap, nil
}
