package model

type Backend interface {
	ConfigType() ConfigType

	// script generation

	SupportsGenerateBackup() bool
	SupportsGenerateGeneric() bool

	GenerateBackup(path string, outputPath string) error
	GenerateGeneric(path string, outputPath string) error

	// direct invocation

	SupportsBackup() bool
	SupportsQuickCheck() bool
	SupportsRun() bool

	QuickCheck(path string) error
	Run(path string, args []string) error

	Backup(path string, rehashSource bool) error

	SupportsBackupShellScriptDiffCheck() bool

	BackupShellScriptDiffCheck(configFilePath string, shellScriptPath string) error
}

type BackendStruct struct {
	ConfigType func() ConfigType

	// script generation

	GenerateBackup  func(path string, outputPath string) error
	GenerateGeneric func(path string, outputPath string) error

	// direct invocation
	Run        func(path string, args []string) error
	Backup     func(path string) error
	QuickCheck func(path string) error

	// misc

	BackupShellScriptDiffCheckfunc func(configFilePath string, shellScriptPath string) error
}
