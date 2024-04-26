package cmds

import "github.com/jgwest/backup-cli/util"

func AddGenericPrefixNode(nodes *util.TextNodes) {
	prefixNode := nodes.NewPrefixTextNode()
	if nodes.IsWindows() {
		// https://stackoverflow.com/questions/17063947/get-current-batchfile-directory
		prefixNode.Out("@echo off", "setlocal")
		prefixNode.Out("set SCRIPTPATH=\"%~f0\"")
	} else {
		prefixNode.Out("#!/bin/bash", "", "set -eu")
		// https://stackoverflow.com/questions/4774054/reliable-way-for-a-bash-script-to-get-the-full-path-to-itself
		prefixNode.Out("SCRIPTPATH=`realpath -s $0`")
	}
	prefixNode.AddExports("SCRIPTPATH")
}
