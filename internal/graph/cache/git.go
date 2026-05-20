package cache

import (
	"os/exec"
	"strings"
)

// gitHeadSHA returns the HEAD SHA of repo, or "" if git is unavailable or
// repo is not a git checkout.
func gitHeadSHA(repo string) string {
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
