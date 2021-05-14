package check

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jgwest/backup-cli/generate"
	"github.com/jgwest/backup-cli/model"
	"gopkg.in/yaml.v2"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func RunCheck(configFilePath string, shellScriptPath string) error {

	// Process the configurate file
	var out *generate.OutputBuffer
	{
		content, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			return err
		}

		model := model.ConfigFile{}

		err = yaml.Unmarshal(content, &model)
		if err != nil {
			return err
		}

		out, err = generate.ProcessConfig(configFilePath, model, false)
		if err != nil {
			return err
		}
	}

	// Read the existing shell script
	content, err := ioutil.ReadFile(shellScriptPath)
	if err != nil {
		return err
	}

	// Diff the desired output with the existing shell script and report differences
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(content), out.ToString(), false)
	if len(diffs) > 0 {
		fmt.Println(dmp.DiffPrettyText(diffs))
		os.Exit(1)
	}

	return nil
}
