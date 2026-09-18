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
)

func savedSearchRoot(out *bytes.Buffer, server *httptest.Server) *appcli.Command {
	return NewRoot(Dependencies{Config: cliConfigFake{}, Credentials: cliCredentialsFake{}, Out: out, Err: &bytes.Buffer{}, HTTPClient: server.Client(), Version: "test"})
}

func TestSavedSearchViewCommandResolvesAndPagesIssues(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		query     string
		issueSkip string
		issueTop  string
		plain     bool
	}{
		{name: "direct id default", args: []string{"youtrack", "saved-search", "view", "51-33", "--url", "SERVER", "--token", "secret", "--json"}, query: "project: APP", issueSkip: "0", issueTop: "50"},
		{name: "named explicit", args: []string{"youtrack", "saved-search", "view", "Release blockers", "--offset", "3", "--limit", "2", "--url", "SERVER", "--token", "secret", "--json"}, query: "project: RELEASE", issueSkip: "3", issueTop: "2"},
		{name: "direct all plain", args: []string{"youtrack", "saved-search", "view", "51-33", "--all", "--url", "SERVER", "--token", "secret", "--plain"}, query: "project: APP", issueSkip: "0", issueTop: "50", plain: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			issueCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodGet, r.Method)
				switch r.URL.Path {
				case "/api/savedQueries/51-33":
					assert.Equal(t, "id,name,query,owner(id,login,fullName)", r.URL.Query().Get("fields"))
					_, _ = w.Write([]byte(`{"id":"51-33","name":"Mine","query":"project: APP","owner":null}`))
				case "/api/savedQueries/Release blockers":
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error_description":"missing"}`))
				case "/api/savedQueries":
					if r.URL.Query().Get("$skip") == "0" {
						_, _ = w.Write([]byte(`[{"id":"51-34","name":"Release blockers","query":"project: RELEASE","owner":null}]`))
						return
					}
					_, _ = w.Write([]byte(`[]`))
				case "/api/issues":
					assert.Equal(t, tc.query, r.URL.Query().Get("query"))
					assert.Equal(t, tc.issueTop, r.URL.Query().Get("$top"))
					if tc.plain {
						issueCalls++
						assert.Equal(t, []string{"0", "1"}[issueCalls-1], r.URL.Query().Get("$skip"))
						if issueCalls == 1 {
							_, _ = w.Write([]byte(`[{"id":"2-1","idReadable":"TT-1","summary":"first","project":{"id":"0-1","name":"Tools","shortName":"TT"},"created":0,"updated":0,"resolved":null}]`))
							return
						}
					} else {
						assert.Equal(t, tc.issueSkip, r.URL.Query().Get("$skip"))
					}
					_, _ = w.Write([]byte(`[]`))
				default:
					t.Fatalf("unexpected request: %s", r.URL.String())
				}
			}))
			defer server.Close()
			args := append([]string(nil), tc.args...)
			for i, arg := range args {
				if arg == "SERVER" {
					args[i] = server.URL
				}
			}
			var out bytes.Buffer
			err := savedSearchRoot(&out, server).Run(context.Background(), args)
			require.NoError(t, err)
			assert.Positive(t, calls)
			if tc.plain {
				assert.Equal(t, "TT-1\tfirst\n", out.String())
				return
			}
			assert.JSONEq(t, `{"entityId":"`+map[bool]string{true: "51-34", false: "51-33"}[tc.name == "named explicit"]+`","name":"`+map[bool]string{true: "Release blockers", false: "Mine"}[tc.name == "named explicit"]+`","query":"`+tc.query+`","owner":null,"issues":[]}`, out.String())
		})
	}
}

func TestSavedSearchViewCommandRejectsInvalidRequestsBeforeHTTP(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "arity", args: []string{"youtrack", "saved-search", "view"}, want: "usage: youtrack saved-search view <saved-search> [--limit <n>] [--offset <n>] [--all]"},
		{name: "blank", args: []string{"youtrack", "saved-search", "view", "   "}, want: "saved search reference must not be blank"},
		{name: "negative offset", args: []string{"youtrack", "saved-search", "view", "Mine", "--offset", "-1"}, want: "offset must be non-negative"},
		{name: "non-positive limit", args: []string{"youtrack", "saved-search", "view", "Mine", "--limit", "0"}, want: "limit must be positive"},
		{name: "all explicit limit", args: []string{"youtrack", "saved-search", "view", "Mine", "--all", "--limit", "50"}, want: "--all and --limit are mutually exclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
			defer server.Close()
			var out bytes.Buffer
			err := savedSearchRoot(&out, server).Run(context.Background(), tc.args)
			require.Error(t, err)
			assert.Equal(t, app.ErrorValidation, app.KindOf(err))
			assert.Equal(t, tc.want, err.Error())
			assert.Zero(t, calls)
		})
	}
}
