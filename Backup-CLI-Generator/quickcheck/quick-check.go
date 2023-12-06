package quickcheck

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func RunQuickCheck(path string) error {

	model, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	if err := processConfig(path, model); err != nil {
		return err
	}

	return nil

}

func processConfig(configFilePath string, config model.ConfigFile) error {

	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	if configType == model.Kopia {
		return fmt.Errorf("unimplemented")

	} else if configType == model.Restic {
		resticQuickCheck(config)
	} else {
		return fmt.Errorf("unimplemented")
	}

	return nil
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
