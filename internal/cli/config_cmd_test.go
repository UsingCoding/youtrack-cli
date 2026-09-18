package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/config"
)

type commandConfigStore struct {
	cfg              config.Config
	values           map[string]string
	setKey, setValue string
}

func (s *commandConfigStore) Load() (config.Config, error)   { return s.cfg, nil }
func (s *commandConfigStore) Save(v config.Config) error     { s.cfg = v; return nil }
func (s *commandConfigStore) Get(key string) (string, error) { return s.values[key], nil }
func (s *commandConfigStore) Set(key, value string) error {
	s.setKey, s.setValue = key, value
	return nil
}

func TestConfigCommandsListGetSetAndJSON(t *testing.T) {
	store := &commandConfigStore{cfg: config.Config{Current: "company", Profiles: map[string]config.Profile{"zebra": {URL: "https://z.example"}, "company": {URL: "https://c.example"}}}, values: map[string]string{"current": "company"}}
	out := &bytes.Buffer{}
	root := NewRoot(Dependencies{Config: store, Credentials: cliCredentialsFake{}, Out: out, Err: &bytes.Buffer{}, Version: "test"})
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "config", "list"}))
	assert.Equal(t, "* company\thttps://c.example\n  zebra\thttps://z.example\n", out.String())
	out.Reset()
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "config", "get", "current"}))
	assert.Equal(t, "company\n", out.String())
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "config", "set", "profile.company.url", "https://new.example"}))
	assert.Equal(t, "profile.company.url", store.setKey)
	assert.Equal(t, "https://new.example", store.setValue)
	out.Reset()
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "config", "list", "--json"}))
	assert.Contains(t, out.String(), `"Current": "company"`)
}

func TestVersionAndInputHelpers(t *testing.T) {
	out := &bytes.Buffer{}
	root := NewRoot(Dependencies{Config: cliConfigFake{}, Credentials: cliCredentialsFake{}, Out: out, Err: &bytes.Buffer{}, Version: "1.2.3", Commit: "abc", BuildDate: "today"})
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "version"}))
	assert.Contains(t, out.String(), "youtrack version 1.2.3")
	out.Reset()
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "version", "--plain"}))
	assert.Equal(t, "1.2.3\n", out.String())
	out.Reset()
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "version", "--json"}))
	assert.Contains(t, out.String(), `"version": "1.2.3"`)
	prompt := &bytes.Buffer{}
	line, err := readLine(strings.NewReader("  value \n"), prompt, "Value: ")
	require.NoError(t, err)
	assert.Equal(t, "value", line)
	assert.Equal(t, "Value: ", prompt.String())
	secret, err := readSecret(strings.NewReader(" value \n"), &bytes.Buffer{}, "")
	require.NoError(t, err)
	assert.Equal(t, "value", secret)
}
