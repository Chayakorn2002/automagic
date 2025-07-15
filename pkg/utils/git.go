package utils

import "os/exec"

func GetCurrentGitCommit() string {
	commit, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return string(commit)
}
