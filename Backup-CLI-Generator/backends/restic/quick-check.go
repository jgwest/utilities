package restic

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func (r ResticBackend) SupportsQuickCheck() bool {
	return true
}

func (r ResticBackend) QuickCheck(path string) error {

	config, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	if configType != model.Restic {
		return fmt.Errorf("configuration file does not support restic")
	}

	return resticQuickCheck(config)

}

func resticQuickCheck(config model.ConfigFile) error {

	invocParams, err := util.GenerateResticDirectInvocation(config)
	if err != nil {
		return err
	}

	env := invocParams.EnvironmentVariables

	envList := os.Environ()
	for k, v := range env {
		envList = append(envList, k+"="+v)
	}

	args := invocParams.Args

	args = append(args, "check")

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = envList
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err = cmd.Run(); err != nil {
		log.Fatal(err)
	}

	return nil
}
