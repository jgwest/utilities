package check

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jgwest/backup-cli/generate"
	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util"
	"gopkg.in/yaml.v2"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func RunCheck(configFilePath string, shellScriptPath string) error {

	// Process the configuration file
	var out *util.OutputBuffer
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

	// If the diffs contain at least one non-equal diff
	containsNonEqual := false
	for _, diff := range diffs {
		if diff.Type != diffmatchpatch.DiffEqual {
			containsNonEqual = true
			break
		}
	}

	if containsNonEqual {
		fmt.Println()
		fmt.Println("ERROR: Mismatch detected:")
		fmt.Println(dmp.DiffPrettyText(diffs))
		os.Exit(1)
	}

	return nil
}
