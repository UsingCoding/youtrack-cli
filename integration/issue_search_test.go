package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type issueSearchRequest struct {
	Method string
	Path   string
	Query  string
	Skip   string
	Top    string
}

func TestBuiltCLIIssueSearch(t *testing.T) {
	query := "project: {Tools} #unresolved -State: Done sort by: updated desc"
	var mu sync.Mutex
	var requests []issueSearchRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/issues", r.URL.Path)
		mu.Lock()
		requests = append(requests, issueSearchRequest{
			Method: r.Method, Path: r.URL.Path, Query: r.URL.Query().Get("query"), Skip: r.URL.Query().Get("$skip"), Top: r.URL.Query().Get("$top"),
		})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("query") {
		case query:
			_, _ = w.Write([]byte(`[
				{"id":"2-9","idReadable":"TT-9","summary":"first from server","project":{"id":"0-1","name":"Tools","shortName":"TT"},"created":1700000000000,"updated":1700000002000,"resolved":null},
				{"id":"2-1","idReadable":"TT-1","summary":"second from server","project":{"id":"0-1","name":"Tools","shortName":"TT"},"created":1700000001000,"updated":1700000003000,"resolved":1700000004000}
			]`))
		case "invalid query":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error_description":"invalid query"}`))
		default:
			http.Error(w, `{"error":"unexpected query"}`, http.StatusBadRequest)
		}
	}))
	defer server.Close()

	binary := filepath.Join(t.TempDir(), "youtrack")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "../cmd/youtrack") // #nosec G204 -- The test builds its own temporary binary.
	build.Dir = "."
	build.Stderr = &bytes.Buffer{}
	require.NoError(t, build.Run())
	configHome := t.TempDir()
	baseEnv := append(os.Environ(), "XDG_CONFIG_HOME="+configHome)

	command := exec.CommandContext(t.Context(), binary, "issue", "search", query, "--offset", "7", "--limit", "2", "--url", server.URL, "--token", "secret-token", "--json") // #nosec G204 -- The test executes its own temporary binary.
	command.Env = baseEnv
	stdout, err := command.Output()
	require.NoError(t, err)

	var output []map[string]any
	require.NoError(t, json.Unmarshal(stdout, &output))
	require.Len(t, output, 2)
	assert.Equal(t, "TT-9", output[0]["id"])
	assert.Equal(t, "TT-1", output[1]["id"])
	assert.Nil(t, output[0]["resolved"])
	assert.Equal(t, time.UnixMilli(1700000004000).Format(time.RFC3339), output[1]["resolved"])
	assert.Equal(t, []issueSearchRequest{{Method: http.MethodGet, Path: "/api/issues", Query: query, Skip: "7", Top: "2"}}, requests)

	invalid := exec.CommandContext(t.Context(), binary, "issue", "search", "invalid query", "--url", server.URL, "--token", "secret-token", "--json") // #nosec G204 -- The test executes its own temporary binary.
	invalid.Env = baseEnv
	var stderr bytes.Buffer
	invalid.Stderr = &stderr
	err = invalid.Run()
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, stderr.String(), "invalid query")
	assert.NotContains(t, stderr.String(), "secret-token")
}
