package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jgwest/backup-cli/model"
)

func FixWindowsPathSuffix(input string) string {

	if strings.HasSuffix(input, "\\\"") {
		input = input[0 : len(input)-2]
		input += "\\\\\""
	}
	return input
}

// expand returns the input string, replacing $var with config file substitutions, or env vars, in that order.
func Expand(input string, configFileSubstitutions []model.Substitution) (output string, err error) {

	substitutions := map[string]string{}

	for _, substitution := range configFileSubstitutions {
		substitutions[substitution.Name] = substitution.Value
	}

	output = os.Expand(input, func(key string) string {

		if val, contains := substitutions[key]; contains {
			return val
		}

		if value, contains := os.LookupEnv(key); contains {
			return value
		}

		if err == nil {
			err = fmt.Errorf("unable to find value for '%s'", key)
		}

		return ""

	})

	return
}

type DirectInvocation struct {
	Args                 []string
	EnvironmentVariables map[string]string
}

func (di DirectInvocation) Execute() error {

	fmt.Println("Environment Variables:")
	envList := os.Environ()
	for k, v := range di.EnvironmentVariables {
		fmt.Println("-", k+"="+v)
		envList = append(envList, k+"="+v)
	}

	fmt.Println()

	fmt.Println("Command Arguments:")
	for _, arg := range di.Args {
		fmt.Println("-", arg)
	}
	fmt.Println()

	cmd := exec.Command(di.Args[0], di.Args[1:]...)
	cmd.Env = envList
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil

}
