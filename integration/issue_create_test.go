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

type createRequest struct {
	Method string
	Path   string
	Body   map[string]any
}

func TestBuiltCLIIssueCreateResolvesThenPostsOnce(t *testing.T) {
	var mu sync.Mutex
	var requests []createRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		mu.Lock()
		requests = append(requests, createRequest{Method: r.Method, Path: r.URL.Path, Body: body})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/admin/projects/APP":
			_, _ = w.Write([]byte(`{"id":"0-1","name":"App","shortName":"APP"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/admin/projects/0-1/customFields":
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"pf-labels","canBeEmpty":true,"field":{"id":"f-labels","name":"Labels","fieldType":{"id":"enum[*]","valueType":"enum","isMultiValue":true}},"bundle":{"id":"labels"}},{"id":"pf-assignee","canBeEmpty":true,"field":{"id":"f-assignee","name":"Assignee","fieldType":{"id":"user[1]","valueType":"user","isMultiValue":false}},"bundle":{"id":"users"}},{"id":"pf-title","canBeEmpty":true,"field":{"id":"f-title","name":"Title","fieldType":{"id":"string","valueType":"string","isMultiValue":false}}},{"id":"pf-priority","canBeEmpty":true,"field":{"id":"f-priority","name":"Priority","fieldType":{"id":"enum[1]","valueType":"enum","isMultiValue":false}},"bundle":{"id":"priority"}}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/admin/projects/0-1/customFields/pf-labels/bundle/values":
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"label-frontend","name":"frontend","archived":false},{"id":"label-api","name":"api","archived":false}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/admin/projects/0-1/customFields/pf-priority/bundle/values":
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"priority-critical","name":"Critical","archived":false}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/admin/customFieldSettings/bundles/user/users/aggregatedUsers":
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"u-1","login":"agent","fullName":"Agent"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/me":
			_, _ = w.Write([]byte(`{"id":"u-1","login":"agent","fullName":"Agent"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"tag-backend","name":"backend"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/issues":
			if body["summary"] == "503" {
				http.Error(w, `{"error":"unavailable"}`, http.StatusServiceUnavailable)
				return
			}
			_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"APP-1","summary":"Login fails after token rotation","description":"line one\n\tline two\n","created":1,"updated":2,"project":{"id":"0-1","name":"App","shortName":"APP"},"tags":[{"id":"tag-backend","name":"backend"}],"customFields":[{"id":"pf-labels","name":"Labels","$type":"MultiEnumIssueCustomField","value":[{"id":"label-frontend","name":"frontend"},{"id":"label-api","name":"api"}]}]}`))
		default:
			t.Fatalf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
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
		cmd := exec.CommandContext(t.Context(), binary, args...) // #nosec G204 -- Executes fixed test-built binary arguments.
		cmd.Env = env
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stdout, err := cmd.Output()
		return stdout, stderr.String(), err
	}

	description := filepath.Join(t.TempDir(), "description.txt")
	descriptionBytes := "line one\n\tline two\n"
	require.NoError(t, os.WriteFile(description, []byte(descriptionBytes), 0o600))
	stdout, stderr, err := run("issue", "create", "APP", "--summary", "Login fails after token rotation", "--description-file", description, "--field", "Labels=frontend", "--field", "Labels=api", "--field", "Assignee=@me", "--field", "Title=left=right,still-literal", "--tag", "backend", "--tag", "backend", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	assert.NotContains(t, stderr, "secret-token")
	var issue map[string]any
	require.NoError(t, json.Unmarshal(stdout, &issue))
	assert.Equal(t, "APP-1", issue["id"])
	assert.NotContains(t, string(stdout), "$type")

	mu.Lock()
	validRequests := append([]createRequest(nil), requests...)
	mu.Unlock()
	require.NotEmpty(t, validRequests)
	post := validRequests[len(validRequests)-1]
	require.Equal(t, createRequest{Method: http.MethodPost, Path: "/api/issues"}, createRequest{Method: post.Method, Path: post.Path})
	assert.Equal(t, "0-1", post.Body["project"].(map[string]any)["id"])
	assert.Equal(t, "Login fails after token rotation", post.Body["summary"])
	assert.Equal(t, descriptionBytes, post.Body["description"])
	fields := post.Body["customFields"].([]any)
	require.Len(t, fields, 3)
	assert.Equal(t, "Labels", fields[0].(map[string]any)["name"])
	assert.NotContains(t, fields[0].(map[string]any), "id")
	assert.Equal(t, "label-frontend", fields[0].(map[string]any)["value"].([]any)[0].(map[string]any)["id"])
	assert.Equal(t, "u-1", fields[1].(map[string]any)["value"].(map[string]any)["id"])
	assert.Equal(t, "left=right,still-literal", fields[2].(map[string]any)["value"])
	assert.Equal(t, "tag-backend", post.Body["tags"].([]any)[0].(map[string]any)["id"])

	_, _, err = run("issue", "create", "APP", "--summary", "Late invalid", "--field", "Priority=missing", "--url", server.URL, "--token", "secret-token")
	require.Error(t, err)
	mu.Lock()
	postsAfterInvalid := 0
	for _, request := range requests[len(validRequests):] {
		if request.Method == http.MethodPost && request.Path == "/api/issues" {
			postsAfterInvalid++
		}
	}
	mu.Unlock()
	assert.Zero(t, postsAfterInvalid)

	_, stderr, err = run("issue", "create", "APP", "--summary", "503", "--url", server.URL, "--token", "secret-token", "--debug")
	require.Error(t, err)
	assert.NotContains(t, stderr, "secret-token")
	mu.Lock()
	posts503 := 0
	for _, request := range requests {
		if request.Method == http.MethodPost && request.Path == "/api/issues" && request.Body["summary"] == "503" {
			posts503++
		}
	}
	mu.Unlock()
	assert.Equal(t, 1, posts503)
}
