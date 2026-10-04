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

type workItemRequest struct {
	Method, Path, Skip string
	Body               map[string]any
}

func TestBuiltCLIIssueTime(t *testing.T) {
	var mu sync.Mutex
	requests := make([]workItemRequest, 0)
	duration := int64(90)
	item := func() map[string]any {
		return map[string]any{"id": "115-7", "date": int64(1790640000000), "duration": map[string]any{"minutes": duration}, "text": "Implement token refresh", "author": map[string]any{"id": "u-alice", "login": "alice", "fullName": "Alice"}, "creator": map[string]any{"id": "u-me", "login": "me", "fullName": "Me"}, "type": map[string]any{"id": "development", "name": "Development"}, "created": int64(1790683200000), "updated": nil}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := workItemRequest{Method: r.Method, Path: r.URL.Path, Skip: r.URL.Query().Get("$skip")}
		if r.Method == http.MethodPost {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request.Body))
		}
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/issues/APP-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "2-1", "idReadable": "APP-1", "project": map[string]any{"id": "0-1"}, "tags": []any{}, "customFields": []any{}})
		case "GET /api/issues/APP-1/sprints":
			_, _ = w.Write([]byte(`[]`))
		case "GET /api/admin/projects/0-1/timeTrackingSettings":
			_, _ = w.Write([]byte(`{"enabled":true}`))
		case "GET /api/admin/projects/0-1/timeTrackingSettings/workItemTypes":
			switch r.URL.Query().Get("$skip") {
			case "0":
				values := make([]map[string]any, 42)
				for i := range values {
					values[i] = map[string]any{"id": "t", "name": "Other"}
				}
				_ = json.NewEncoder(w).Encode(values)
			case "42":
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "development", "name": "Development"}})
			case "43":
				_, _ = w.Write([]byte(`[]`))
			default:
				t.Fatalf("unexpected type page %s", r.URL.String())
			}
		case "GET /api/users":
			switch r.URL.Query().Get("$skip") {
			case "0":
				values := make([]map[string]any, 42)
				for i := range values {
					values[i] = map[string]any{"id": "u", "login": "other"}
				}
				_ = json.NewEncoder(w).Encode(values)
			case "42":
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "u-alice", "login": "alice", "fullName": "Alice"}})
			case "43":
				_, _ = w.Write([]byte(`[]`))
			default:
				t.Fatalf("unexpected user page %s", r.URL.String())
			}
		case "GET /api/issues/2-1/timeTracking/workItems":
			_ = json.NewEncoder(w).Encode([]map[string]any{item()})
		case "GET /api/issues/2-1/timeTracking/workItems/115-7":
			_ = json.NewEncoder(w).Encode(item())
		case "POST /api/issues/2-1/timeTracking/workItems":
			_ = json.NewEncoder(w).Encode(item())
		case "POST /api/issues/2-1/timeTracking/workItems/115-7":
			duration = 120
			updated := item()
			updated["text"] = ""
			updated["type"] = nil
			_ = json.NewEncoder(w).Encode(updated)
		case "DELETE /api/issues/2-1/timeTracking/workItems/115-7":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
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
		cmd := exec.CommandContext(t.Context(), binary, args...) // #nosec G204 -- Executes the test-built binary with fixed test arguments.
		cmd.Env = env
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stdout, err := cmd.Output()
		return stdout, stderr.String(), err
	}
	stdout, stderr, err := run("issue", "time", "types", "APP-1", "--all", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	assert.NotContains(t, stderr, "secret-token")
	var types []map[string]any
	require.NoError(t, json.Unmarshal(stdout, &types))
	require.Len(t, types, 43)
	stdout, _, err = run("issue", "time", "add", "APP-1", "--duration", "1h30m", "--date", "2026-09-29", "--type", "Development", "--text", "Implement token refresh", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	var added map[string]any
	require.NoError(t, json.Unmarshal(stdout, &added))
	assert.Equal(t, float64(90), added["durationMinutes"])
	_, _, err = run("issue", "time", "list", "APP-1", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	_, _, err = run("issue", "time", "view", "APP-1", "115-7", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	stdout, _, err = run("issue", "time", "edit", "APP-1", "115-7", "--duration", "2h", "--text", "", "--clear-type", "--author", "alice", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	var edited map[string]any
	require.NoError(t, json.Unmarshal(stdout, &edited))
	assert.Equal(t, float64(120), edited["durationMinutes"])
	assert.Nil(t, edited["type"])
	_, _, err = run("issue", "time", "remove", "APP-1", "115-7", "--yes", "--url", server.URL, "--token", "secret-token")
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	var create, edit *workItemRequest
	deletes := 0
	for i := range requests {
		request := &requests[i]
		if request.Method == http.MethodPost && request.Path == "/api/issues/2-1/timeTracking/workItems" {
			create = request
		}
		if request.Method == http.MethodPost && request.Path == "/api/issues/2-1/timeTracking/workItems/115-7" {
			edit = request
		}
		if request.Method == http.MethodDelete && request.Path == "/api/issues/2-1/timeTracking/workItems/115-7" {
			deletes++
		}
	}
	require.NotNil(t, create)
	assert.Equal(t, float64(1790640000000), create.Body["date"])
	require.NotNil(t, edit)
	assert.Equal(t, map[string]any{"duration": map[string]any{"minutes": float64(120)}, "text": "", "type": nil, "author": map[string]any{"id": "u-alice"}}, edit.Body)
	assert.Equal(t, 1, deletes)
}
