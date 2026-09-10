package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreRoundTripWithBurntSushiTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "youtrack", "config.toml")
	store, err := NewStore(path)
	require.NoError(t, err)

	want := Config{Current: "company", Profiles: map[string]Profile{"company": {URL: "https://example.youtrack.cloud"}}}
	require.NoError(t, store.Save(want))
	got, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, want, got)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

type testCreds map[string]string

func (c testCreds) Get(profile string) (string, error) { return c[profile], nil }

func TestResolveDoesNotReuseStoredTokenForURLOverride(t *testing.T) {
	t.Setenv("YOUTRACK_URL", "")
	t.Setenv("YOUTRACK_TOKEN", "")
	t.Setenv("YOUTRACK_PROFILE", "")
	cfg := Config{Current: "company", Profiles: map[string]Profile{"company": {URL: "https://old.example"}}}

	_, err := Resolve(cfg, testCreds{"company": "stored-secret"}, RuntimeInput{URL: "https://new.example"})
	require.Error(t, err)

	got, err := Resolve(cfg, testCreds{"company": "stored-secret"}, RuntimeInput{URL: "https://new.example", Token: "explicit"})
	require.NoError(t, err)
	assert.Equal(t, "https://new.example", got.URL)
	assert.Equal(t, "explicit", got.Token)
}
