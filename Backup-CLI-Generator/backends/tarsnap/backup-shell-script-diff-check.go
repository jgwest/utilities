package tarsnap

import (
	"os"

	"github.com/jgwest/backup-cli/model"
	diffgeneratedbackupscript "github.com/jgwest/backup-cli/util/cmds/diff-generated-backup-script"
	"gopkg.in/yaml.v2"
)

func (r TarsnapBackend) SupportsBackupShellScriptDiffCheck() bool {
	return true
}

func (r TarsnapBackend) BackupShellScriptDiffCheck(configFilePath string, shellScriptPath string) error {

	// Process the configuration file
	content, err := os.ReadFile(configFilePath)
	if err != nil {
		return err
	}

	model := model.ConfigFile{}

	err = yaml.Unmarshal(content, &model)
	if err != nil {
		return err
	}

	generatedBackupShellScriptContents, err := processGenerateBackupConfig(configFilePath, model, false)
	if err != nil {
		return err
	}

	return diffgeneratedbackupscript.DiffGeneratedBackupShellScript(generatedBackupShellScriptContents, shellScriptPath)

}
