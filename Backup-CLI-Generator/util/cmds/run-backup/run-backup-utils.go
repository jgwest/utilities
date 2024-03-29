package runbackup

type BackupRunObject struct {
	BackupDateTime string

	GlobalExcludes []string

	RobocopyFileExcludes   []string
	RobocopyFolderExcludes []string

	Todo []string
}
