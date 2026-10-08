package integration

import (
	"bytes"
	"encoding/json"
	"io"
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

type linkRequest struct {
	Method string
	Path   string
	Body   map[string]any
}

func TestBuiltCLIIssueLinks(t *testing.T) {
	var mu sync.Mutex
	requests := make([]linkRequest, 0)
	parentPresent := false
	customPresent := false
	types := make([]map[string]any, 42)
	for i := range types {
		types[i] = map[string]any{"id": "filler-" + string(rune('a'+i)), "name": "Filler", "directed": false, "aggregation": false, "sourceToTarget": "filler"}
	}
	types = append(types,
		map[string]any{"id": "parent", "name": "Subtask", "directed": true, "aggregation": true, "sourceToTarget": "parent for", "targetToSource": "subtask of"},
		map[string]any{"id": "custom", "name": "Custom", "directed": false, "aggregation": false, "sourceToTarget": "relates to"},
	)
	issue := func(id, readable string) map[string]any { return map[string]any{"id": id, "idReadable": readable} }
	summary := func(id, readable string) map[string]any {
		return map[string]any{"id": id, "idReadable": readable, "summary": readable, "project": map[string]any{"id": "0-1", "name": "App", "shortName": "APP"}, "created": 0, "updated": 0}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Body != nil {
			data, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			if len(data) > 0 {
				require.NoError(t, json.Unmarshal(data, &body))
			}
		}
		mu.Lock()
		requests = append(requests, linkRequest{Method: r.Method, Path: r.URL.Path, Body: body})
		defer mu.Unlock()
		switch r.Method + " " + r.URL.Path {
		case "GET /api/issueLinkTypes":
			switch r.URL.Query().Get("$skip") {
			case "0":
				_ = json.NewEncoder(w).Encode(types[:42])
			case "42":
				_ = json.NewEncoder(w).Encode(types[42:])
			default:
				_, _ = w.Write([]byte(`[]`))
			}
		case "GET /api/issues/APP-CHILD":
			_ = json.NewEncoder(w).Encode(issue("2-child", "APP-CHILD"))
		case "GET /api/issues/APP-PARENT":
			_ = json.NewEncoder(w).Encode(issue("2-parent", "APP-PARENT"))
		case "GET /api/issues/2-child/links/parentt/issues":
			if parentPresent && r.URL.Query().Get("$skip") == "0" {
				_ = json.NewEncoder(w).Encode([]map[string]any{summary("2-parent", "APP-PARENT")})
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case "GET /api/issues/2-parent/links/parents/issues":
			if parentPresent && r.URL.Query().Get("$skip") == "0" {
				_ = json.NewEncoder(w).Encode([]map[string]any{summary("2-child", "APP-CHILD")})
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case "POST /api/issues/2-child/links/parentt/issues":
			assert.Equal(t, map[string]any{"id": "2-parent"}, body)
			parentPresent = true
			w.WriteHeader(http.StatusNoContent)
		case "DELETE /api/issues/2-child/links/parentt/issues/2-parent":
			parentPresent = false
			w.WriteHeader(http.StatusNoContent)
		case "GET /api/issues/2-child/links/custom/issues":
			if customPresent && r.URL.Query().Get("$skip") == "0" {
				_ = json.NewEncoder(w).Encode([]map[string]any{summary("2-parent", "APP-PARENT")})
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case "POST /api/issues/2-child/links/custom/issues":
			assert.Equal(t, map[string]any{"id": "2-parent"}, body)
			customPresent = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	binary := filepath.Join(t.TempDir(), "youtrack")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "../cmd/youtrack") // #nosec G204 -- Builds the repository binary in a test-only temporary path.
	build.Dir = "."
	build.Stderr = &bytes.Buffer{}
	require.NoError(t, build.Run())
	env := append(os.Environ(), "XDG_CONFIG_HOME="+t.TempDir())
	run := func(args ...string) ([]byte, string, error) {
		cmd := exec.CommandContext(t.Context(), binary, args...) // #nosec G204 -- Executes the test-built binary with test-controlled arguments.
		cmd.Env = env
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		return stdout.Bytes(), stderr.String(), err
	}

	stdout, _, err := run("issue", "link", "types", "--all", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	var discovered []map[string]any
	require.NoError(t, json.Unmarshal(stdout, &discovered))
	assert.Len(t, discovered, 44)

	stdout, _, err = run("issue", "link", "add", "APP-CHILD", "APP-PARENT", "--type", "subtask of", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	var added map[string]any
	require.NoError(t, json.Unmarshal(stdout, &added))
	assert.Equal(t, "APP-CHILD", added["issueId"])
	assert.Equal(t, "APP-PARENT", added["targetIssueId"])
	assert.Equal(t, "inward", added["relation"].(map[string]any)["direction"])
	assert.Equal(t, true, added["present"])
	assert.Equal(t, true, added["changed"])

	for _, args := range [][]string{
		{"issue", "link", "list", "APP-CHILD", "--type", "subtask of", "--all", "--url", server.URL, "--token", "secret-token", "--json"},
		{"issue", "link", "list", "APP-PARENT", "--type", "parent for", "--all", "--url", server.URL, "--token", "secret-token", "--json"},
		{"issue", "link", "remove", "APP-CHILD", "APP-PARENT", "--type", "subtask of", "--url", server.URL, "--token", "secret-token", "--json"},
		{"issue", "link", "add", "APP-CHILD", "APP-PARENT", "--type", "relates to", "--url", server.URL, "--token", "secret-token", "--json"},
		{"issue", "link", "list", "APP-CHILD", "--type", "relates to", "--all", "--url", server.URL, "--token", "secret-token", "--json"},
	} {
		_, _, err = run(args...)
		require.NoError(t, err)
	}

	mu.Lock()
	defer mu.Unlock()
	postParent, deleteParent, postCustom, issueDeletes := 0, 0, 0, 0
	for _, request := range requests {
		switch request.Method + " " + request.Path {
		case "POST /api/issues/2-child/links/parentt/issues":
			postParent++
		case "DELETE /api/issues/2-child/links/parentt/issues/2-parent":
			deleteParent++
		case "POST /api/issues/2-child/links/custom/issues":
			postCustom++
		}
		if request.Method == http.MethodDelete && request.Path != "/api/issues/2-child/links/parentt/issues/2-parent" {
			issueDeletes++
		}
	}
	assert.Equal(t, 1, postParent)
	assert.Equal(t, 1, deleteParent)
	assert.Equal(t, 1, postCustom)
	assert.Zero(t, issueDeletes)
}
