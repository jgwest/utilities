package runbackup

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

type BackupRunObject struct {
	BackupDateTime string

	GlobalExcludes []string

	RobocopyFileExcludes   []string
	RobocopyFolderExcludes []string

	Todo []string
}

func GetCurrentTimeTag() (string, error) {

	isWindows := runtime.GOOS == "windows"

	if isWindows {
		cmd := exec.Command("cmd", "/c", "echo %DATE%-%TIME:~1%")
		var out strings.Builder
		cmd.Stdout = &out

		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}

		res := out.String()
		res = strings.ReplaceAll(res, "\r", "")
		res = strings.ReplaceAll(res, "\n", "")
		res = strings.TrimSpace(res)

		return res, nil
	} else {
		// 	backupDateTime.Out("BACKUP_DATE_TIME=`date +%F_%H:%M:%S`")

		return "", fmt.Errorf("linux is unsupported")
	}

}
