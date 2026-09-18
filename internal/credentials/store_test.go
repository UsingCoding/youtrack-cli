package credentials

import (
	"crypto/subtle"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreReadsWritesDeletesAndPersistsSecurely(t *testing.T) {
	path := filepath.Join(t.TempDir(), "youtrack", "credentials.toml")
	store, err := NewStore(path)
	require.NoError(t, err)
	missing, err := store.Get("company")
	require.NoError(t, err)
	assert.Empty(t, missing)

	const token = "test-token"
	require.NoError(t, store.Set("company", token))
	got, err := store.Get("company")
	require.NoError(t, err)
	assert.Equal(t, 1, subtle.ConstantTimeCompare([]byte(token), []byte(got)))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "[profiles.company]")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	require.NoError(t, store.Delete("company"))
	got, err = store.Get("company")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestStoreNormalizesNilProfilesAndRejectsMalformedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.toml")
	require.NoError(t, os.WriteFile(path, []byte("# no profiles\n"), 0o600))
	store, err := NewStore(path)
	require.NoError(t, err)
	got, err := store.Get("company")
	require.NoError(t, err)
	assert.Empty(t, got)
	require.NoError(t, store.Set("company", "test-token"))
	require.NoError(t, os.WriteFile(path, []byte("profiles = ["), 0o600))
	_, err = store.Get("company")
	require.Error(t, err)
}

func TestStoreUsesXDGDefaultPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	path, err := DefaultPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "youtrack", "credentials.toml"), path)
	store, err := NewStore("")
	require.NoError(t, err)
	assert.Equal(t, path, store.Path)
	assert.True(t, strings.HasSuffix(store.Path, "credentials.toml"))
}
