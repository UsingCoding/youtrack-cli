package skill

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	youtrackcli "github.com/UsingCoding/youtrack-cli"
)

func withProjectDirectory(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	project := t.TempDir()
	require.NoError(t, os.Chdir(project))
	t.Cleanup(func() { require.NoError(t, os.Chdir(cwd)) })
	return project
}

func TestTargetsInventory(t *testing.T) {
	targets, err := Targets()
	require.NoError(t, err)
	got := make([]string, 0, len(targets))
	for _, target := range targets {
		got = append(got, target.Name)
		assert.NotEmpty(t, target.ProjectRoot)
	}
	assert.Equal(t, []string{"claude", "codex", "cursor", "opencode", "agents"}, got)
}

func TestProjectSkillStatusInstallAndRemove(t *testing.T) {
	project := withProjectDirectory(t)
	preexisting := filepath.Join(project, ".codex", "skills", "youtrack-cli")
	require.NoError(t, os.MkdirAll(preexisting, 0o755))
	statuses, err := List(true)
	require.NoError(t, err)
	var codex Status
	for _, status := range statuses {
		if status.Target == "codex" {
			codex = status
		}
	}
	assert.True(t, codex.Exists)
	assert.Equal(t, preexisting, codex.Path)

	installed, err := Install(true, "codex")
	require.NoError(t, err)
	assert.Equal(t, []string{preexisting}, installed)
	actual, err := os.ReadFile(filepath.Join(preexisting, "SKILL.md"))
	require.NoError(t, err)
	expected, err := youtrackcli.SkillsFS.ReadFile("skills/youtrack-cli/SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
	assert.True(t, sort.StringsAreSorted(installed))

	removed, err := Remove(true, "codex")
	require.NoError(t, err)
	assert.Equal(t, []string{preexisting}, removed)
	removed, err = Remove(true, "codex")
	require.NoError(t, err)
	assert.Empty(t, removed)
}

func TestProjectInstallHandlesUnknownAndDefaultTarget(t *testing.T) {
	project := withProjectDirectory(t)
	_, err := Install(true, "missing")
	require.Error(t, err)

	installed, err := Install(true, "")
	require.NoError(t, err)
	want := filepath.Join(project, ".agents", "skills", "youtrack-cli")
	assert.Equal(t, []string{want}, installed)
	_, err = os.Stat(filepath.Join(want, "references"))
	require.NoError(t, err)
}
