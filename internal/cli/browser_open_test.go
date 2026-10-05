package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appcli "github.com/urfave/cli/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/config"
)

type browserOpenerFake struct {
	urls []string
	err  error
}

func (f *browserOpenerFake) Open(rawURL string) error {
	f.urls = append(f.urls, rawURL)
	return f.err
}

type credentialReadFail struct{ calls int }

func (f *credentialReadFail) Get(string) (string, error) {
	f.calls++
	return "", errors.New("credentials must not be read")
}
func (*credentialReadFail) Set(string, string) error { return nil }
func (*credentialReadFail) Delete(string) error      { return nil }

type countingConfigStore struct{ calls int }

func (f *countingConfigStore) Load() (config.Config, error) {
	f.calls++
	return config.Config{Profiles: map[string]config.Profile{}}, nil
}
func (*countingConfigStore) Save(config.Config) error   { return nil }
func (*countingConfigStore) Get(string) (string, error) { return "", nil }
func (*countingConfigStore) Set(string, string) error   { return nil }

func browserRoot(out *bytes.Buffer, deps Dependencies) *appcli.Command {
	deps.Out = out
	deps.Err = &bytes.Buffer{}
	deps.Version = "test"
	return NewRoot(deps)
}

func TestIssueSearchOpenUsesURLOnlyRuntime(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
		t.Fatal("browser search must not make HTTP requests")
	}))
	defer server.Close()
	out := &bytes.Buffer{}
	credentials := &credentialReadFail{}
	opener := &browserOpenerFake{}

	err := browserRoot(out, Dependencies{Config: cliConfigFake{}, Credentials: credentials, BrowserOpener: opener, HTTPClient: server.Client()}).Run(context.Background(), []string{
		"youtrack", "issue", "search", "project: APP #Open", "open", "--url", server.URL + "/youtrack", "--print-url", "--plain",
	})

	require.NoError(t, err)
	assert.Equal(t, server.URL+"/youtrack/issues?q=project%3A+APP+%23Open\n", out.String())
	assert.Zero(t, credentials.calls)
	assert.Zero(t, requests)
	assert.Empty(t, opener.urls)
}

func TestIssueSearchOpenSuffixAndLiteralOpenRemainDistinct(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.Equal(t, "/api/issues", r.URL.Path)
		require.Equal(t, "open", r.URL.Query().Get("query"))
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	literalOut := &bytes.Buffer{}
	err := browserRoot(literalOut, Dependencies{Config: cliConfigFake{}, Credentials: cliCredentialsFake{}, BrowserOpener: &browserOpenerFake{}, HTTPClient: server.Client()}).Run(context.Background(), []string{
		"youtrack", "issue", "search", "open", "--url", server.URL, "--token", "secret", "--json",
	})
	require.NoError(t, err)
	assert.Equal(t, "[]\n", literalOut.String())
	assert.Equal(t, 1, requests)

	suffixOut := &bytes.Buffer{}
	err = browserRoot(suffixOut, Dependencies{Config: cliConfigFake{}, Credentials: &credentialReadFail{}, BrowserOpener: &browserOpenerFake{}, HTTPClient: server.Client()}).Run(context.Background(), []string{
		"youtrack", "issue", "search", "open", "open", "--url", server.URL, "--print-url", "--plain",
	})
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/issues?q=open\n", suffixOut.String())
	assert.Equal(t, 1, requests)
}

func TestIssueSearchOpenRejectsInvalidModesBeforeConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "unexpected suffix", args: []string{"youtrack", "issue", "search", "query", "OPEN"}},
		{name: "extra positional", args: []string{"youtrack", "issue", "search", "query", "open", "extra"}},
		{name: "pagination", args: []string{"youtrack", "issue", "search", "query", "open", "--limit", "1"}},
		{name: "print URL without suffix", args: []string{"youtrack", "issue", "search", "query", "--print-url"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			cfg := &countingConfigStore{}
			opener := &browserOpenerFake{}

			err := browserRoot(out, Dependencies{Config: cfg, Credentials: &credentialReadFail{}, BrowserOpener: opener}).Run(context.Background(), tc.args)

			require.Error(t, err)
			assert.Equal(t, app.ErrorValidation, app.KindOf(err))
			assert.Zero(t, cfg.calls)
			assert.Empty(t, opener.urls)
			assert.Empty(t, out.String())
		})
	}
}

func TestBrowserOpenDispatchAndFailureContracts(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "JSON", args: []string{"--json"}, want: "{\n  \"url\": \"https://youtrack.example/issues?q=project%3A+APP\",\n  \"opened\": true\n}\n"},
		{name: "plain", args: []string{"--plain"}, want: "https://youtrack.example/issues?q=project%3A+APP\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			opener := &browserOpenerFake{}
			args := append([]string{"youtrack", "issue", "search", "project: APP", "open", "--url", "https://youtrack.example"}, tc.args...)

			err := browserRoot(out, Dependencies{Config: cliConfigFake{}, Credentials: &credentialReadFail{}, BrowserOpener: opener}).Run(context.Background(), args)

			require.NoError(t, err)
			assert.Equal(t, []string{"https://youtrack.example/issues?q=project%3A+APP"}, opener.urls)
			assert.Equal(t, tc.want, out.String())
		})
	}
	t.Run("failure has no success output", func(t *testing.T) {
		out := &bytes.Buffer{}
		opener := &browserOpenerFake{err: errors.New("launcher missing")}

		err := browserRoot(out, Dependencies{Config: cliConfigFake{}, Credentials: &credentialReadFail{}, BrowserOpener: opener}).Run(context.Background(), []string{
			"youtrack", "issue", "search", "project: APP", "open", "--url", "https://youtrack.example", "--json",
		})

		require.Error(t, err)
		assert.Equal(t, app.ErrorRuntime, app.KindOf(err))
		assert.Contains(t, err.Error(), "--print-url")
		assert.Empty(t, out.String())
	})
}

func TestIssueAndSavedSearchOpenUseLightweightMetadata(t *testing.T) {
	t.Run("issue", func(t *testing.T) {
		calls := make([]string, 0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, r.URL.Path)
			require.Equal(t, "/api/issues/alias", r.URL.Path)
			require.Equal(t, "id,idReadable", r.URL.Query().Get("fields"))
			_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"APP-1"}`))
		}))
		defer server.Close()
		out := &bytes.Buffer{}

		err := browserRoot(out, Dependencies{Config: cliConfigFake{}, Credentials: cliCredentialsFake{}, BrowserOpener: &browserOpenerFake{}, HTTPClient: server.Client()}).Run(context.Background(), []string{
			"youtrack", "issue", "open", "alias", "--url", server.URL, "--token", "secret", "--print-url", "--plain",
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"/api/issues/alias"}, calls)
		assert.Equal(t, server.URL+"/issue/APP-1\n", out.String())
	})
	t.Run("saved search direct ID", func(t *testing.T) {
		calls := make([]string, 0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, r.URL.Path)
			require.Equal(t, "/api/savedQueries/51-33", r.URL.Path)
			_, _ = w.Write([]byte(`{"id":"51-33","name":"Mine","query":"project: APP"}`))
		}))
		defer server.Close()
		out := &bytes.Buffer{}

		err := browserRoot(out, Dependencies{Config: cliConfigFake{}, Credentials: cliCredentialsFake{}, BrowserOpener: &browserOpenerFake{}, HTTPClient: server.Client()}).Run(context.Background(), []string{
			"youtrack", "saved-search", "open", "51-33", "--url", server.URL, "--token", "secret", "--print-url", "--plain",
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"/api/savedQueries/51-33"}, calls)
		assert.Equal(t, server.URL+"/issues?q=project%3A+APP\n", out.String())
	})
	t.Run("saved search name", func(t *testing.T) {
		calls := make([]string, 0)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, r.URL.Path)
			switch r.URL.Path {
			case "/api/savedQueries/Mine":
				http.NotFound(w, r)
			case "/api/savedQueries":
				if r.URL.Query().Get("$skip") == "0" {
					_, _ = w.Write([]byte(`[{"id":"51-33","name":"Mine","query":"project: APP"}]`))
					return
				}
				_, _ = w.Write([]byte(`[]`))
			default:
				t.Fatalf("unexpected request: %s", r.URL.Path)
			}
		}))
		defer server.Close()
		out := &bytes.Buffer{}

		err := browserRoot(out, Dependencies{Config: cliConfigFake{}, Credentials: cliCredentialsFake{}, BrowserOpener: &browserOpenerFake{}, HTTPClient: server.Client()}).Run(context.Background(), []string{
			"youtrack", "saved-search", "open", "Mine", "--url", server.URL, "--token", "secret", "--print-url", "--plain",
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"/api/savedQueries/Mine", "/api/savedQueries", "/api/savedQueries"}, calls)
		assert.Equal(t, server.URL+"/issues?q=project%3A+APP\n", out.String())
	})
}
