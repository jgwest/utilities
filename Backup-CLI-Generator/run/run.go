package run

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func Run(path string, args []string) error {

	model, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	if err := processConfig(args, model); err != nil {
		return err
	}

	return nil

}

func processConfig(userArgs []string, config model.ConfigFile) error {

	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	if configType == model.Restic {
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

		cmdArgs := args[1:]

		cmdArgs = append(cmdArgs, userArgs...)

		cmd := exec.Command(args[0], cmdArgs...)
		cmd.Env = envList
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err = cmd.Run(); err != nil {
			log.Fatal(err)
		}

		return nil

	} else {
		return fmt.Errorf("unsupported type")
	}

}
