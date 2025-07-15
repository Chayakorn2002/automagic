package version

import (
	"fmt"

	"github.com/bilbo290/automagic/pkg/utils"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

func PrintVersionInfo() {
	// TODO: get version from config

	Commit = utils.GetCurrentGitCommit()

	fmt.Printf("automagic Multi-Provider Automation (GitLab & GitHub)\n")
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Commit: %s\n", Commit)
}
