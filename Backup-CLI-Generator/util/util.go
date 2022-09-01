package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jgwest/backup-cli/model"
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/yaml.v2"
)

func DiffMissingFields(content []byte) (err error) {

	convertToInterfaceAndBack := func(content []byte) (mapString string, err error) {

		// Convert to string => interface
		mapStringToIntr := map[string]interface{}{}
		if err = yaml.Unmarshal(content, &mapStringToIntr); err != nil {
			return
		}

		// Convert back to string
		var out []byte
		if out, err = yaml.Marshal(mapStringToIntr); err != nil {
			return
		}
		mapString = string(out)

		return
	}

	var mapString string
	if mapString, err = convertToInterfaceAndBack(content); err != nil {
		return
	}

	var structString string
	{
		// Convert string -> ConfigFile
		model := model.ConfigFile{}
		if err = yaml.Unmarshal(content, &model); err != nil {
			return
		}

		// Convert ConfigFile -> string
		var out []byte
		if out, err = yaml.Marshal(model); err != nil {
			return
		}
		if structString, err = convertToInterfaceAndBack(out); err != nil {
			return
		}
	}

	// Compare the two
	{
		dmp := diffmatchpatch.New()

		diffs := dmp.DiffMain(mapString, structString, false)

		nonequalDiffs := []diffmatchpatch.Diff{}

		for index, currDiff := range diffs {
			if currDiff.Type != diffmatchpatch.DiffEqual {
				nonequalDiffs = append(nonequalDiffs, diffs[index])
			}
		}

		if len(nonequalDiffs) > 0 {

			fmt.Println()
			fmt.Println("-------")
			fmt.Println(dmp.DiffPrettyText(diffs))
			fmt.Println("-------")
			return errors.New("diffs reported")
		}
	}

	return nil
}

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
