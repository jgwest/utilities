package generic

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"gopkg.in/yaml.v2"
)

func RunGeneric(path string, outputPath string) error {

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	// Look for invalid fields in the YAML
	if err := util.DiffMissingFields(content); err != nil {
		return err
	}

	model := model.ConfigFile{}
	if err = yaml.Unmarshal(content, &model); err != nil {
		return err
	}

	result, err := ProcessConfig(path, model, false)
	if err != nil {
		return err
	}

	// If the output path already exists, don't overwrite it
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("output path already exists: %s", outputPath)
	}

	if err := ioutil.WriteFile(outputPath, []byte(result.ToString()), 0700); err != nil {
		return err
	}

	fmt.Println(result.ToString())

	return nil

}

func ProcessConfig(configFilePath string, config model.ConfigFile, dryRun bool) (*util.OutputBuffer, error) {

	configType, err := config.GetConfigType()
	if err != nil {
		return nil, err
	}

	buffer := util.OutputBuffer{
		IsWindows: runtime.GOOS == "windows",
	}

	if buffer.IsWindows {
		buffer.Lines = []string{"@echo off", "setlocal"}
		// https://stackoverflow.com/questions/17063947/get-current-batchfile-directory
		buffer.Out("set SCRIPTPATH=\"%~f0\"")
	} else {
		buffer.Lines = []string{"#!/bin/bash", "", "set -eu"}
		// https://stackoverflow.com/questions/4774054/reliable-way-for-a-bash-script-to-get-the-full-path-to-itself
		buffer.Out("SCRIPTPATH=`realpath -s $0`")
	}

	if configType == model.Restic {
		err = resticGenerateGenericInvocation(config, &buffer)
	} else if configType == model.Kopia {
		err = kopiaGenerateGenericInvocation(config, &buffer)
	} else {
		return nil, fmt.Errorf("unsupported configType: %v", configType)
	}

	if err != nil {
		return nil, err
	}

	return &buffer, nil
}

func kopiaGenerateGenericInvocation(config model.ConfigFile, buffer *util.OutputBuffer) error {

	kopiaCredentials, err := config.GetKopiaCredential()
	if err != nil {
		return err
	}

	if kopiaCredentials.S3 == nil || kopiaCredentials.KopiaS3 == nil {
		return fmt.Errorf("missing S3 credentials")
	}

	if kopiaCredentials.S3.AccessKeyID == "" || kopiaCredentials.S3.SecretAccessKey == "" {
		return fmt.Errorf("missing S3 credential values")
	}

	if kopiaCredentials.KopiaS3.Bucket == "" || kopiaCredentials.KopiaS3.Region == "" || kopiaCredentials.KopiaS3.Endpoint == "" {
		return fmt.Errorf("missing S3 credential values")
	}

	if kopiaCredentials.Password == "" {
		return fmt.Errorf("missing kopia password")
	}

	buffer.Out()
	buffer.Header("Credentials ")
	buffer.SetEnv("AWS_ACCESS_KEY_ID", kopiaCredentials.S3.AccessKeyID)
	buffer.SetEnv("AWS_SECRET_ACCESS_KEY", kopiaCredentials.S3.SecretAccessKey)

	if len(kopiaCredentials.Password) > 0 {
		buffer.SetEnv("KOPIA_PASSWORD", kopiaCredentials.Password)
	}

	buffer.Out()
	buffer.Header("Connect repository")

	cliInvocation := fmt.Sprintf("kopia repository connect s3 --bucket=\"%s\" --access-key=\"%s\" --secret-access-key=\"%s\" --password=\"%s\" --endpoint=\"%s\" --region=\"%s\"",
		kopiaCredentials.KopiaS3.Bucket,
		buffer.Env("AWS_ACCESS_KEY_ID"),
		buffer.Env("AWS_SECRET_ACCESS_KEY"),
		buffer.Env("KOPIA_PASSWORD"),
		kopiaCredentials.KopiaS3.Endpoint,
		kopiaCredentials.KopiaS3.Region)

	buffer.Out(cliInvocation)

	buffer.Out()
	buffer.Header("Invoke generic command")

	if buffer.IsWindows {
		cliInvocation = "kopia %*"
		buffer.Out(cliInvocation)
	} else {
		cliInvocation = "kopia $*"
		buffer.Out("bash -c \"" + cliInvocation + "\"")
	}

	return nil
}

func resticGenerateGenericInvocation(config model.ConfigFile, buffer *util.OutputBuffer) error {

	resticCredential, err := config.GetResticCredential()
	if err != nil {
		return err
	}

	if resticCredential.S3 != nil {
		buffer.Out()
		buffer.Header("Credentials")
		buffer.SetEnv("AWS_ACCESS_KEY_ID", resticCredential.S3.AccessKeyID)
		buffer.SetEnv("AWS_SECRET_ACCESS_KEY", resticCredential.S3.SecretAccessKey)
	}

	if len(resticCredential.Password) > 0 && len(resticCredential.PasswordFile) > 0 {
		return errors.New("both password and password file are specified")
	}

	if len(resticCredential.Password) > 0 {
		buffer.SetEnv("RESTIC_PASSWORD", resticCredential.Password)

	} else if len(resticCredential.PasswordFile) > 0 {
		buffer.SetEnv("RESTIC_PASSWORD_FILE", resticCredential.PasswordFile)

	} else {
		return errors.New("no restic password found")
	}
	url := ""
	if resticCredential.S3 != nil {
		url = "s3:" + resticCredential.S3.URL
	} else if resticCredential.RESTEndpoint != "" {
		url = "rest:" + resticCredential.RESTEndpoint
	} else {
		return errors.New("unable to locate connection credentials")
	}

	cacertSubstring := ""
	if resticCredential.CACert != "" {
		expandedPath, err := util.Expand(resticCredential.CACert, config.Substitutions)
		if err != nil {
			return err
		}
		cacertSubstring = "--cacert \"" + expandedPath + "\" "
	}

	additionalParams := ""
	if buffer.IsWindows {
		additionalParams = "%*"

	} else {
		additionalParams = "$*"
	}

	cliInvocation := fmt.Sprintf("restic -r %s --verbose %s %s",
		url,
		cacertSubstring,
		additionalParams)

	buffer.Out()

	if buffer.IsWindows {
		buffer.Out(cliInvocation)
	} else {
		buffer.Out("bash -c \"" + cliInvocation + "\"")
	}

	return nil
}
