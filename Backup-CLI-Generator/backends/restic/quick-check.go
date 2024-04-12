package restic

import (
	"log"
	"os"
	"os/exec"

	"github.com/jgwest/backup-cli/model"
)

func (r ResticBackend) SupportsQuickCheck() bool {
	return true
}

func (r ResticBackend) QuickCheck(path string) error {

	config, err := extractAndValidateConfigFile(path)
	if err != nil {
		return err
	}

	return resticQuickCheck(config)

}

func resticQuickCheck(config model.ConfigFile) error {

	invocParams, err := generateResticDirectInvocation(config)
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
