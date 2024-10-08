package restic

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
)

func (r ResticBackend) SupportsRun() bool {
	return true
}

func (r ResticBackend) Run(path string, args []string) error {
	model, err := model.ReadConfigFile(path)
	if err != nil {
		return err
	}

	if err := processConfigRun(args, model); err != nil {
		return err
	}

	return nil

}

func processConfigRun(userArgs []string, config model.ConfigFile) error {

	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	if configType != model.Restic {
		return fmt.Errorf("unsupported type: %v", configType)
	}

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

}
