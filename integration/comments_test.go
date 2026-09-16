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

type commentRequest struct {
	Method, Path, Skip, Top string
	Body                    map[string]any
}

func TestBuiltCLIComments(t *testing.T) {
	var mu sync.Mutex
	requests := make([]commentRequest, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := commentRequest{Method: r.Method, Path: r.URL.Path, Skip: r.URL.Query().Get("$skip"), Top: r.URL.Query().Get("$top")}
		if r.Method == http.MethodPost {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request.Body))
		}
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/issues/NOTFOUND/comments":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_description":"missing"}`))
		case r.URL.Path == "/api/issues/FORBIDDEN/comments":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error_description":"forbidden"}`))
		case r.URL.Path == "/api/issues/APP-503/comments" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusServiceUnavailable)
		case r.URL.Path == "/api/issues/APP-1/comments" && r.Method == http.MethodGet:
			switch r.URL.Query().Get("$skip") {
			case "0":
				items := make([]map[string]any, 42)
				for i := range items {
					items[i] = map[string]any{"id": "4-" + string(rune('a'+i%26)), "text": nil, "author": nil, "created": 0, "updated": nil, "deleted": i == 0}
				}
				require.NoError(t, json.NewEncoder(w).Encode(items))
			case "42":
				require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{{"id": "4-last", "text": "final", "author": nil, "created": 1, "updated": nil, "deleted": false}}))
			case "43":
				_, _ = w.Write([]byte(`[]`))
			default:
				t.Fatalf("unexpected comment page: %s", r.URL.String())
			}
		case r.URL.Path == "/api/issues/APP-1/comments" && r.Method == http.MethodPost:
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "4-created", "text": request.Body["text"], "author": nil, "created": 0, "updated": nil, "deleted": false}))
		case r.URL.Path == "/api/issues/APP-1/comments/4-edit" && r.Method == http.MethodPost:
			require.Equal(t, "id,text,author(id,login,fullName),created,updated,deleted", r.URL.Query().Get("fields"))
			require.Len(t, r.URL.Query(), 1)
			require.Equal(t, map[string]any{"text": "revised\n\tcomment\n"}, request.Body)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "4-edit", "text": request.Body["text"], "author": nil, "created": 0, "updated": nil, "deleted": false}))
		case r.URL.Path == "/api/issues/APP-1/comments/4-remove" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)
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
		cmd := exec.CommandContext(t.Context(), binary, args...) // #nosec G204 -- Executes the test-built binary with fixed test arguments.
		cmd.Env = env
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stdout, err := cmd.Output()
		return stdout, stderr.String(), err
	}

	stdout, stderr, err := run("issue", "comment", "list", "APP-1", "--all", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	assert.NotContains(t, stderr, "secret-token")
	var listed []map[string]any
	require.NoError(t, json.Unmarshal(stdout, &listed))
	require.Len(t, listed, 43)
	assert.Equal(t, "4-a", listed[0]["entityId"])
	assert.Equal(t, true, listed[0]["deleted"])
	assert.Nil(t, listed[0]["text"])
	assert.Equal(t, "4-last", listed[42]["entityId"])

	inline := "inline\n\tcontent\n"
	_, _, err = run("issue", "comment", "add", "APP-1", "--text", inline, "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	file := filepath.Join(t.TempDir(), "comment.txt")
	fileText := "file\ncontent\n"
	require.NoError(t, os.WriteFile(file, []byte(fileText), 0o600))
	_, _, err = run("issue", "comment", "add", "APP-1", "--file", file, "--url", server.URL, "--token", "secret-token", "--plain")
	require.NoError(t, err)

	editText := "revised\n\tcomment\n"
	stdout, _, err = run("issue", "comment", "edit", "APP-1", "4-edit", "--text", editText, "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	var edited map[string]any
	require.NoError(t, json.Unmarshal(stdout, &edited))
	assert.Equal(t, "4-edit", edited["entityId"])
	assert.Equal(t, editText, edited["text"])
	assert.Nil(t, edited["author"])
	assert.Nil(t, edited["updated"])
	assert.Equal(t, false, edited["deleted"])

	stdout, _, err = run("issue", "comment", "remove", "APP-1", "4-remove", "--url", server.URL, "--token", "secret-token", "--json")
	require.NoError(t, err)
	assert.JSONEq(t, `{"entityId":"4-remove","removed":true}`, string(stdout))
	stdout, _, err = run("issue", "comment", "remove", "APP-1", "4-remove", "--url", server.URL, "--token", "secret-token", "--plain")
	require.NoError(t, err)
	assert.Empty(t, stdout)

	for _, tc := range []struct {
		issue string
		code  int
	}{{"NOTFOUND", 4}, {"FORBIDDEN", 3}} {
		_, errOut, runErr := run("issue", "comment", "list", tc.issue, "--url", server.URL, "--token", "secret-token", "--debug")
		require.Error(t, runErr)
		var exit *exec.ExitError
		require.ErrorAs(t, runErr, &exit)
		assert.Equal(t, tc.code, exit.ExitCode())
		assert.NotContains(t, errOut, "secret-token")
		assert.NotContains(t, errOut, "Authorization")
	}
	_, errOut, runErr := run("issue", "comment", "add", "APP-503", "--text", "once", "--url", server.URL, "--token", "secret-token", "--debug")
	require.Error(t, runErr)
	assert.NotContains(t, errOut, "secret-token")

	mu.Lock()
	defer mu.Unlock()
	var addBodies []map[string]any
	var edits []commentRequest
	removes := 0
	attempts503 := 0
	for _, request := range requests {
		if request.Path == "/api/issues/APP-1/comments" && request.Method == http.MethodPost {
			addBodies = append(addBodies, request.Body)
		}
		if request.Path == "/api/issues/APP-1/comments/4-edit" {
			edits = append(edits, request)
		}
		if request.Path == "/api/issues/APP-1/comments/4-remove" {
			removes++
			assert.Equal(t, map[string]any{"deleted": true}, request.Body)
		}
		if request.Path == "/api/issues/APP-503/comments" {
			attempts503++
		}
	}
	assert.Equal(t, []map[string]any{{"text": inline}, {"text": fileText}}, addBodies)
	assert.Equal(t, 2, removes)
	assert.Equal(t, 1, attempts503)
	assert.Equal(t, []commentRequest{{Method: http.MethodGet, Path: "/api/issues/APP-1/comments", Skip: "0", Top: "50"}, {Method: http.MethodGet, Path: "/api/issues/APP-1/comments", Skip: "42", Top: "50"}, {Method: http.MethodGet, Path: "/api/issues/APP-1/comments", Skip: "43", Top: "50"}}, requests[:3])
	assert.Equal(t, []commentRequest{{Method: http.MethodPost, Path: "/api/issues/APP-1/comments/4-edit", Body: map[string]any{"text": editText}}}, edits)
}
