package gitversion_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/contracttesting/cli/internal/gitversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

func TestResolve_InGitRepoWithCommit_ReturnsShortSHA(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init")
	gitIn(t, dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "initial")
	expected := gitIn(t, dir, "rev-parse", "--short", "HEAD")

	got, err := gitversion.Resolve(dir)

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestResolve_OutsideGitRepo_ReturnsInstructiveError(t *testing.T) {
	dir := t.TempDir()

	_, err := gitversion.Resolve(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pass --version")
}
