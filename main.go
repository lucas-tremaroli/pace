package main

import (
	"github.com/lucas-tremaroli/pace/v2/cmd"
	"github.com/lucas-tremaroli/pace/v2/cmd/mcp"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	mcp.SetVersion(version)
	cmd.Execute()
}
