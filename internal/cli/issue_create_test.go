package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func TestIssueCreateUsesNestedGlobalsAndPreservesDescriptions(t *testing.T) {
	fileDescription := "from file\n\twith whitespace\n"
	path := filepath.Join(t.TempDir(), "description.txt")
	require.NoError(t, os.WriteFile(path, []byte(fileDescription), 0o600))

	for _, tc := range []struct {
		name, description, output string
		args                      []string
	}{
		{"inline json", "  inline\n\ttext  ", "json", []string{"youtrack", "issue", "create", "APP", "--summary", "New", "--description", "  inline\n\ttext  ", "--url", "SERVER", "--token", "secret", "--json"}},
		{"file plain", fileDescription, "plain", []string{"youtrack", "issue", "create", "APP", "--summary", "New", "--description-file", path, "--url", "SERVER", "--token", "secret", "--plain"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch r.URL.Path {
				case "/api/admin/projects/APP":
					assert.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"id":"0-1","name":"App","shortName":"APP"}`))
				case "/api/issues":
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
					var payload map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					assert.Equal(t, tc.description, payload["description"])
					_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"APP-1","summary":"New","description":"","created":1,"updated":2,"project":{"id":"0-1","name":"App","shortName":"APP"},"tags":[],"customFields":[]}`))
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			out := &bytes.Buffer{}
			err := issueSearchRoot(out, server).Run(context.Background(), replaceServer(tc.args, server.URL))

			require.NoError(t, err)
			assert.Equal(t, 2, calls)
			if tc.output == "plain" {
				assert.Equal(t, "APP-1\tNew\n", out.String())
				return
			}
			var issue map[string]any
			require.NoError(t, json.Unmarshal(out.Bytes(), &issue))
			assert.Equal(t, "APP-1", issue["id"])
			assert.NotContains(t, out.String(), "$type")
		})
	}
}

func TestIssueCreateValidatesLocally(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"missing project", []string{"youtrack", "issue", "create", "--summary", "New"}},
		{"extra project", []string{"youtrack", "issue", "create", "APP", "OTHER", "--summary", "New"}},
		{"missing summary", []string{"youtrack", "issue", "create", "APP"}},
		{"blank summary", []string{"youtrack", "issue", "create", "APP", "--summary", " \t"}},
		{"both descriptions", []string{"youtrack", "issue", "create", "APP", "--summary", "New", "--description", "x", "--description-file", "missing"}},
		{"malformed field", []string{"youtrack", "issue", "create", "APP", "--summary", "New", "--field", "Priority"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
			defer server.Close()

			err := issueSearchRoot(&bytes.Buffer{}, server).Run(context.Background(), tc.args)

			require.Error(t, err)
			assert.Equal(t, app.ErrorValidation, app.KindOf(err))
			assert.Zero(t, calls)
		})
	}
}
