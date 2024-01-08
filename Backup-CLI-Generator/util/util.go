package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jgwest/backup-cli/model"
)

type OutputBuffer struct {
	IsWindows bool
	Lines     []string
}

func (buffer *OutputBuffer) ToString() string {
	output := ""

	for _, line := range buffer.Lines {

		output += line

		if buffer.IsWindows {
			output += "\r\n"
		} else {
			output += "\n"
		}
	}

	return output
}

func FixWindowsPathSuffix(input string) string {

	if strings.HasSuffix(input, "\\\"") {
		input = input[0 : len(input)-2]
		input += "\\\\\""
	}
	return input
}

func (buffer *OutputBuffer) SetEnv(envName string, value string) {
	if buffer.IsWindows {

		value = FixWindowsPathSuffix(value)
		buffer.Out(fmt.Sprintf("set %s=%s", envName, value))
	} else {
		// Export is used due to need to use 'bash -c (...)' at end of script
		buffer.Out(fmt.Sprintf("export %s=\"%s\"", envName, value))
	}
}

func (buffer *OutputBuffer) Env(envName string) string {
	if buffer.IsWindows {
		return "%" + envName + "%"
	} else {
		return "${" + envName + "}"
	}
}

func (buffer *OutputBuffer) Header(str string) {

	if !strings.HasSuffix(str, " ") {
		str += " "
	}

	for len(str) < 80 {
		str = str + "-"
	}

	if buffer.IsWindows {
		buffer.Out("REM " + str)
	} else {
		buffer.Out("# " + str)
	}

}

// func (buffer *OutputBuffer) comment(str string) {
// 	if buffer.isWindows {
// 		buffer.Out("REM " + str)
// 	} else {
// 		buffer.Out("# " + str)
// 	}

// }

func (buffer *OutputBuffer) Out(str ...string) {
	if len(str) == 0 {
		str = []string{""}
	}

	buffer.Lines = append(buffer.Lines, str...)
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

func FindConfigFile() (string, error) {

	fileinfoList, err := ioutil.ReadDir(".")
	if err != nil {
		return "", err
	}

	matches := []string{}

	for _, info := range fileinfoList {

		if info.IsDir() {
			continue
		}

		if !strings.HasSuffix(strings.ToLower(info.Name()), ".yaml") {
			continue
		}

		matches = append(matches, info.Name())
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple YAML files in folder")
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no YAML files in folder")
	}

	return matches[0], nil

}
