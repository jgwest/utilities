package generic

import (
	"fmt"
	"io/ioutil"

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

	return fmt.Errorf("unimplemented")

}
