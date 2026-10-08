package integration

import (
	"bytes"
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

func buildBrowserHandoffBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "youtrack")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "../cmd/youtrack") // #nosec G204 -- The test builds its own temporary binary.
	build.Dir = "."
	build.Stderr = &bytes.Buffer{}
	require.NoError(t, build.Run())
	return binary
}

func TestBuiltCLIDirectSearchOpenPrintsContextURLWithoutCredentialsOrHTTP(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
		t.Fatal("direct browser search must not make HTTP requests")
	}))
	defer server.Close()
	binary := buildBrowserHandoffBinary(t)
	command := exec.CommandContext(t.Context(), binary, "issue", "search", "--url", server.URL+"/youtrack", "--print-url", "--plain", "--", "project: APP #Open&+%{}\"✓\n", "open") // #nosec G204 -- The test executes its own temporary binary.
	command.Env = append(os.Environ(), "XDG_CONFIG_HOME="+t.TempDir())

	stdout, err := command.Output()

	require.NoError(t, err)
	assert.Equal(t, server.URL+"/youtrack/issues?q=project%3A+APP+%23Open%26%2B%25%7B%7D%22%E2%9C%93%0A\n", string(stdout))
	assert.Zero(t, requests)
}

func TestBuiltCLIAuthenticatedOpenUsesMetadataOnly(t *testing.T) {
	var mu sync.Mutex
	paths := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/api/issues/alias":
			require.Equal(t, "id,idReadable", r.URL.Query().Get("fields"))
			_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"APP-1"}`))
		case "/api/savedQueries/51-33":
			_, _ = w.Write([]byte(`{"id":"51-33","name":"Mine","query":"project: APP"}`))
		default:
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
	}))
	defer server.Close()
	binary := buildBrowserHandoffBinary(t)
	env := append(os.Environ(), "XDG_CONFIG_HOME="+t.TempDir())

	issue := exec.CommandContext(t.Context(), binary, "issue", "open", "alias", "--url", server.URL, "--token", "secret-token", "--print-url", "--plain") // #nosec G204 -- The test executes its own temporary binary.
	issue.Env = env
	issueStdout, err := issue.Output()
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/issue/APP-1\n", string(issueStdout))

	saved := exec.CommandContext(t.Context(), binary, "saved-search", "open", "51-33", "--url", server.URL, "--token", "secret-token", "--print-url", "--plain") // #nosec G204 -- The test executes its own temporary binary.
	saved.Env = env
	savedStdout, err := saved.Output()
	require.NoError(t, err)
	assert.Equal(t, server.URL+"/issues?q=project%3A+APP\n", string(savedStdout))

	assert.Equal(t, []string{"/api/issues/alias", "/api/savedQueries/51-33"}, paths)
}
