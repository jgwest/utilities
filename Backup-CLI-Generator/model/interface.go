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

	Backup(path string) error
}
