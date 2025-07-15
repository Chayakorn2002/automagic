package main

import (
	"github.com/bilbo290/automagic/cmd"
	"github.com/bilbo290/automagic/pkg/version"
)

// Build-time variables (set via ldflags)
var (
	versionString = "dev"
	commit        = "unknown"
)

func init() {
	version.Version = versionString
	version.Commit = commit
}

func main() {
	cmd.Execute()
}
