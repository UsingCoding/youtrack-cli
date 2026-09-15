package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	appcli "github.com/urfave/cli/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/config"
)

type cliConfigFake struct{}

func (cliConfigFake) Load() (config.Config, error) {
	return config.Config{Profiles: map[string]config.Profile{}}, nil
}
func (cliConfigFake) Save(config.Config) error   { return nil }
func (cliConfigFake) Get(string) (string, error) { return "", nil }
func (cliConfigFake) Set(string, string) error   { return nil }

type cliCredentialsFake struct{}

func (cliCredentialsFake) Get(string) (string, error) { return "", nil }
func (cliCredentialsFake) Set(string, string) error   { return nil }
func (cliCredentialsFake) Delete(string) error        { return nil }

func issueSearchRoot(out *bytes.Buffer, server *httptest.Server) *appcli.Command {
	return NewRoot(Dependencies{
		Config: cliConfigFake{}, Credentials: cliCredentialsFake{}, Out: out, Err: &bytes.Buffer{}, HTTPClient: server.Client(), Version: "test",
	})
}

func TestIssueSearchCommandUsesNestedGlobalsAndPagingFlags(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		wantQuery  string
		wantSkip   string
		wantTop    string
		wantOutput string
	}{
		{name: "default", args: []string{"youtrack", "issue", "search", "project: APP", "--url", "SERVER", "--token", "secret", "--json"}, wantQuery: "project: APP", wantSkip: "0", wantTop: "50", wantOutput: "[]\n"},
		{name: "explicit bounded paging", args: []string{"youtrack", "issue", "search", "project: APP", "--limit", "2", "--offset", "3", "--url", "SERVER", "--token", "secret", "--json"}, wantQuery: "project: APP", wantSkip: "3", wantTop: "2", wantOutput: "[]\n"},
		{name: "leading hyphen query", args: []string{"youtrack", "issue", "search", "--url", "SERVER", "--token", "secret", "--json", "--", "-State: Done project: APP"}, wantQuery: "-State: Done project: APP", wantSkip: "0", wantTop: "50", wantOutput: "[]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "/api/issues", r.URL.Path)
				require.Equal(t, tc.wantQuery, r.URL.Query().Get("query"))
				require.Equal(t, tc.wantSkip, r.URL.Query().Get("$skip"))
				require.Equal(t, tc.wantTop, r.URL.Query().Get("$top"))
				require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(`[]`))
			}))
			defer server.Close()
			out := &bytes.Buffer{}
			args := append([]string(nil), tc.args...)
			for i, arg := range args {
				if arg == "SERVER" {
					args[i] = server.URL
				}
			}

			err := issueSearchRoot(out, server).Run(context.Background(), args)

			require.NoError(t, err)
			assert.Equal(t, 1, calls)
			assert.Equal(t, tc.wantOutput, out.String())
		})
	}
}

func TestIssueSearchCommandRejectsInvalidRequestsBeforeHTTP(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "blank query", args: []string{"youtrack", "issue", "search", "   "}, want: "search query must not be blank"},
		{name: "negative offset", args: []string{"youtrack", "issue", "search", "project: APP", "--offset", "-1"}, want: "offset must be non-negative"},
		{name: "explicit default conflicts with all", args: []string{"youtrack", "issue", "search", "project: APP", "--all", "--limit", "50"}, want: "--all and --limit are mutually exclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				t.Fatal("invalid search must not make HTTP requests")
			}))
			defer server.Close()
			out := &bytes.Buffer{}

			err := issueSearchRoot(out, server).Run(context.Background(), tc.args)
			require.Error(t, err)
			assert.Equal(t, app.ErrorValidation, app.KindOf(err))
			assert.Equal(t, tc.want, err.Error())
			assert.Zero(t, calls)
		})
	}
}
