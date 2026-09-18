package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillCommandsUseProjectScope(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	project := t.TempDir()
	require.NoError(t, os.Chdir(project))
	t.Cleanup(func() { require.NoError(t, os.Chdir(cwd)) })
	out := &bytes.Buffer{}
	root := NewRoot(Dependencies{Config: cliConfigFake{}, Credentials: cliCredentialsFake{}, Out: out, Err: &bytes.Buffer{}, Version: "test"})
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "skill", "install", "--project", "--target", "agents"}))
	path := filepath.Join(project, ".agents", "skills", "youtrack-cli")
	assert.Contains(t, out.String(), "Installed youtrack-cli skill: "+path)
	out.Reset()
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "skill", "update", "--project", "--target", "agents"}))
	assert.Contains(t, out.String(), "Updated youtrack-cli skill: "+path)
	out.Reset()
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "skill", "remove", "--project", "--target", "agents"}))
	assert.Contains(t, out.String(), "Removed youtrack-cli skill: "+path)
}
