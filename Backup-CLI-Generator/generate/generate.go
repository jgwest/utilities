package generate

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/jgwest/backup-cli/model"
	"gopkg.in/yaml.v2"
)

func RunGenerate(path string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	model := model.ConfigFile{}

	err = yaml.Unmarshal(content, &model)
	if err != nil {
		return err
	}

	// fmt.Println("args", model)

	return ProcessConfig(model)

}

func ProcessConfig(config model.ConfigFile) error {

	buffer := OutputBuffer{
		lines: []string{},
	}

	// Process Global Excludes
	if len(config.GlobalExcludes) > 0 {
		for index, exclude := range config.GlobalExcludes {

			substring := ""

			if index > 0 {
				substring = "$EXCLUDES "
			}

			buffer.out("EXCLUDES=\"" + substring + "--exclude '" + exclude + "'\"")

		}
	}
	configType, err := config.GetConfigType()
	if err != nil {
		return err
	}

	// Process folders
	if len(config.Folders) == 0 {
		return errors.New("at least one folder is required")
	}

	for index, path := range config.Folders {

		if len(path.Excludes) != 0 && (configType == model.Restic || configType == model.Tarsnap) {
			return fmt.Errorf("backup utility '%s' does not support excludes", configType)
		}
		substring := ""

		if index > 0 {
			substring = "$TODO "
		}

		buffer.out(fmt.Sprintf("TODO=\"%s'%s'\"", substring, path))

	}

	for _, line := range buffer.lines {
		fmt.Println(line)
	}

	return nil
}

type OutputBuffer struct {
	lines []string
}

func (buffer *OutputBuffer) out(str string) {
	buffer.lines = append(buffer.lines, str)
}
