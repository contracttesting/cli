package gitversion

import (
	"errors"
	"os/exec"
	"strings"
)

// ErrNoGitVersion is returned when the version cannot be auto-detected from git.
var ErrNoGitVersion = errors.New("could not detect version from git (not a git repository, or git is not installed): pass --version explicitly")

// Resolve returns the short commit SHA of HEAD in dir, for use as the
// participant version when --version is omitted.
func Resolve(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", ErrNoGitVersion
	}

	return strings.TrimSpace(string(out)), nil
}
