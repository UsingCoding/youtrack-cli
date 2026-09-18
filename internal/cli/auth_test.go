package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/config"
)

type mutableConfigStore struct{ value config.Config }

func (s *mutableConfigStore) Load() (config.Config, error) { return s.value, nil }
func (s *mutableConfigStore) Save(v config.Config) error   { s.value = v; return nil }
func (s *mutableConfigStore) Get(string) (string, error)   { return "", nil }
func (s *mutableConfigStore) Set(string, string) error     { return nil }

type mutableCredentialStore struct {
	values  map[string]string
	deleted string
}

func (s *mutableCredentialStore) Get(profile string) (string, error) { return s.values[profile], nil }
func (s *mutableCredentialStore) Set(profile, token string) error {
	s.values[profile] = token
	return nil
}
func (s *mutableCredentialStore) Delete(profile string) error {
	s.deleted = profile
	delete(s.values, profile)
	return nil
}

func TestAuthCommandsPersistValidateAndReportStatus(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "/api/users/me", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":"u-1","login":"alice","fullName":"Alice"}`))
	}))
	defer server.Close()
	cfg := &mutableConfigStore{value: config.Config{Profiles: map[string]config.Profile{}}}
	creds := &mutableCredentialStore{values: map[string]string{}}
	out := &bytes.Buffer{}
	root := NewRoot(Dependencies{Config: cfg, Credentials: creds, Out: out, Err: &bytes.Buffer{}, HTTPClient: server.Client(), Version: "test"})
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "auth", "login", "--profile", "company", "--url", server.URL, "--token", "test-token"}))
	assert.Equal(t, "company", cfg.value.Current)
	assert.Equal(t, server.URL, cfg.value.Profiles["company"].URL)
	assert.Equal(t, 1, calls)
	assert.NotContains(t, out.String(), "test-token")

	out.Reset()
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "auth", "status", "--json"}))
	assert.Contains(t, out.String(), "\"login\": \"alice\"")
	assert.NotContains(t, out.String(), "test-token")
	require.NoError(t, root.Run(context.Background(), []string{"youtrack", "auth", "logout"}))
	assert.Equal(t, "company", creds.deleted)
}

func TestAuthLoginRejectsBlankCredentialsBeforeNetwork(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	root := NewRoot(Dependencies{Config: &mutableConfigStore{value: config.Config{Profiles: map[string]config.Profile{}}}, Credentials: &mutableCredentialStore{values: map[string]string{}}, In: bytes.NewBufferString("\n"), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, HTTPClient: server.Client(), Version: "test"})
	err := root.Run(context.Background(), []string{"youtrack", "auth", "login", "--url", " "})
	require.Error(t, err)
	assert.Equal(t, app.ErrorValidation, app.KindOf(err))
	assert.Zero(t, calls)
}
