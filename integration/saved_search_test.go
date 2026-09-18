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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savedSearchRequest struct {
	Method string
	Path   string
	Skip   string
	Top    string
	Query  string
}

func TestBuiltCLISavedSearchView(t *testing.T) {
	storedQuery := "project: {Tools} #Unresolved -State: Done sort by: updated desc"
	var mu sync.Mutex
	requests := make([]savedSearchRequest, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, savedSearchRequest{Method: r.Method, Path: r.URL.Path, Skip: r.URL.Query().Get("$skip"), Top: r.URL.Query().Get("$top"), Query: r.URL.Query().Get("query")})
		mu.Unlock()
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/savedQueries/Release blockers":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_description":"missing"}`))
		case "/api/savedQueries":
			switch r.URL.Query().Get("$skip") {
			case "0":
				items := make([]map[string]any, 42)
				for i := range items {
					items[i] = map[string]any{"id": "other", "name": "other", "query": "project: OTHER", "owner": nil}
				}
				require.NoError(t, json.NewEncoder(w).Encode(items))
			case "42":
				require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{{"id": "51-33", "name": "Release blockers", "query": storedQuery, "owner": map[string]any{"id": "1-2", "login": "ada", "fullName": "Ada Lovelace"}}}))
			case "43":
				_, _ = w.Write([]byte(`[]`))
			default:
				t.Fatalf("unexpected saved-query page: %s", r.URL.String())
			}
		case "/api/issues":
			assert.Equal(t, storedQuery, r.URL.Query().Get("query"))
			assert.Equal(t, "7", r.URL.Query().Get("$skip"))
			assert.Equal(t, "2", r.URL.Query().Get("$top"))
			_, _ = w.Write([]byte(`[
				{"id":"2-9","idReadable":"TT-9","summary":"first from server","project":{"id":"0-1","name":"Tools","shortName":"TT"},"created":1700000000000,"updated":1700000002000,"resolved":null},
				{"id":"2-1","idReadable":"TT-1","summary":"second from server","project":{"id":"0-1","name":"Tools","shortName":"TT"},"created":1700000001000,"updated":1700000003000,"resolved":1700000004000}
			]`))
		default:
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
	}))
	defer server.Close()

	binary := filepath.Join(t.TempDir(), "youtrack")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "../cmd/youtrack") // #nosec G204 -- The test builds its own temporary binary.
	build.Dir = "."
	build.Stderr = &bytes.Buffer{}
	require.NoError(t, build.Run())
	command := exec.CommandContext(t.Context(), binary, "saved-search", "view", "Release blockers", "--offset", "7", "--limit", "2", "--url", server.URL, "--token", "secret-token", "--json") // #nosec G204 -- The test executes its own temporary binary.
	command.Env = append(os.Environ(), "XDG_CONFIG_HOME="+t.TempDir())
	stdout, err := command.Output()
	require.NoError(t, err)

	var output map[string]any
	require.NoError(t, json.Unmarshal(stdout, &output))
	assert.Equal(t, "Release blockers", output["name"])
	assert.Equal(t, storedQuery, output["query"])
	assert.Equal(t, map[string]any{"entityId": "1-2", "login": "ada", "name": "Ada Lovelace"}, output["owner"])
	issues := output["issues"].([]any)
	require.Len(t, issues, 2)
	assert.Equal(t, "TT-9", issues[0].(map[string]any)["id"])
	assert.Equal(t, "TT-1", issues[1].(map[string]any)["id"])
	assert.Equal(t, []savedSearchRequest{
		{Method: http.MethodGet, Path: "/api/savedQueries/Release blockers"},
		{Method: http.MethodGet, Path: "/api/savedQueries", Skip: "0", Top: "50"},
		{Method: http.MethodGet, Path: "/api/savedQueries", Skip: "42", Top: "50"},
		{Method: http.MethodGet, Path: "/api/savedQueries", Skip: "43", Top: "50"},
		{Method: http.MethodGet, Path: "/api/issues", Skip: "7", Top: "2", Query: storedQuery},
	}, requests)
}
