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
	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func commentRoot(out *bytes.Buffer, server *httptest.Server) *appcli.Command {
	return issueSearchRoot(out, server)
}

func TestIssueCommentListUsesNestedGlobalsAndPagination(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		skip, top string
	}{
		{"default", []string{"youtrack", "issue", "comment", "list", "APP-1", "--url", "SERVER", "--token", "secret", "--json"}, "0", "50"},
		{"bounded", []string{"youtrack", "issue", "comment", "list", "APP-1", "--limit", "2", "--offset", "3", "--url", "SERVER", "--token", "secret", "--plain"}, "3", "2"},
		{"all", []string{"youtrack", "issue", "comment", "list", "APP-1", "--all", "--url", "SERVER", "--token", "secret", "--plain"}, "0", "50"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "/api/issues/APP-1/comments", r.URL.Path)
				assert.Equal(t, tc.skip, r.URL.Query().Get("$skip"))
				assert.Equal(t, tc.top, r.URL.Query().Get("$top"))
				_, _ = w.Write([]byte(`[]`))
			}))
			defer server.Close()
			args := replaceServer(tc.args, server.URL)
			out := &bytes.Buffer{}
			require.NoError(t, commentRoot(out, server).Run(context.Background(), args))
			assert.Equal(t, 1, calls)
			if tc.name == "default" {
				assert.Equal(t, "[]\n", out.String())
			}
		})
	}
}

func TestIssueCommentAddPreservesInlineAndFileBytes(t *testing.T) {
	fileText := "from file\n\twith whitespace\n"
	path := filepath.Join(t.TempDir(), "comment.txt")
	require.NoError(t, os.WriteFile(path, []byte(fileText), 0o600))
	for _, tc := range []struct {
		name, text string
		args       []string
	}{
		{"inline", "  inline\n\ttext  ", []string{"youtrack", "issue", "comment", "add", "APP-1", "--text", "  inline\n\ttext  ", "--url", "SERVER", "--token", "secret", "--json"}},
		{"file", fileText, []string{"youtrack", "issue", "comment", "add", "APP-1", "--file", path, "--url", "SERVER", "--token", "secret", "--plain"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/issues/APP-1/comments", r.URL.Path)
				var body map[string]string
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, map[string]string{"text": tc.text}, body)
				_, _ = w.Write([]byte(`{"id":"4-1","text":null,"author":null,"created":0,"updated":null,"deleted":false}`))
			}))
			defer server.Close()
			out := &bytes.Buffer{}
			require.NoError(t, commentRoot(out, server).Run(context.Background(), replaceServer(tc.args, server.URL)))
			assert.Equal(t, 1, calls)
		})
	}
}

func TestIssueCommentEditPreservesInlineAndFileBytes(t *testing.T) {
	fileText := "from file\n\twith whitespace\n"
	path := filepath.Join(t.TempDir(), "comment.txt")
	require.NoError(t, os.WriteFile(path, []byte(fileText), 0o600))
	for _, tc := range []struct {
		name, text string
		args       []string
	}{
		{"inline JSON", "  inline\n\ttext  ", []string{"youtrack", "issue", "comment", "edit", "APP-1", "4-1", "--text", "  inline\n\ttext  ", "--url", "SERVER", "--token", "secret", "--json"}},
		{"file plain", fileText, []string{"youtrack", "issue", "comment", "edit", "APP-1", "4-1", "--file", path, "--url", "SERVER", "--token", "secret", "--plain"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/issues/APP-1/comments/4-1", r.URL.Path)
				var body map[string]string
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, map[string]string{"text": tc.text}, body)
				_, _ = w.Write([]byte(`{"id":"4-1","text":null,"author":null,"created":0,"updated":null,"deleted":false}`))
			}))
			defer server.Close()

			out := &bytes.Buffer{}
			require.NoError(t, commentRoot(out, server).Run(context.Background(), replaceServer(tc.args, server.URL)))
			assert.Equal(t, 1, calls)
			if tc.name == "inline JSON" {
				var response map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(out.Bytes(), &response))
				assert.Len(t, response, 6)
				assert.JSONEq(t, `"4-1"`, string(response["entityId"]))
				assert.JSONEq(t, `false`, string(response["deleted"]))
				assert.NotEmpty(t, response["created"])
				assert.JSONEq(t, `null`, string(response["author"]))
				assert.JSONEq(t, `null`, string(response["text"]))
				assert.JSONEq(t, `null`, string(response["updated"]))
			} else {
				assert.Equal(t, "4-1\n", out.String())
			}
		})
	}
}

func TestIssueCommentEditValidatesLocally(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"missing arguments", []string{"youtrack", "issue", "comment", "edit"}, "usage: youtrack issue comment edit <issue> <comment-entity-id> (--text <text> | --file <path>)"},
		{"extra argument", []string{"youtrack", "issue", "comment", "edit", "APP-1", "4-1", "extra", "--text", "x"}, "usage: youtrack issue comment edit <issue> <comment-entity-id> (--text <text> | --file <path>)"},
		{"blank issue", []string{"youtrack", "issue", "comment", "edit", " ", "4-1", "--text", "x"}, "issue reference must not be blank"},
		{"blank comment ID", []string{"youtrack", "issue", "comment", "edit", "APP-1", " ", "--text", "x"}, "comment ID must not be blank"},
		{"no source", []string{"youtrack", "issue", "comment", "edit", "APP-1", "4-1"}, "exactly one of --text or --file is required"},
		{"both sources", []string{"youtrack", "issue", "comment", "edit", "APP-1", "4-1", "--text", "x", "--file", "x"}, "exactly one of --text or --file is required"},
		{"blank text", []string{"youtrack", "issue", "comment", "edit", "APP-1", "4-1", "--text", " \t"}, "comment text must not be blank"},
		{"missing file", []string{"youtrack", "issue", "comment", "edit", "APP-1", "4-1", "--file", "/missing/comment"}, "open /missing/comment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
			defer server.Close()
			err := commentRoot(&bytes.Buffer{}, server).Run(context.Background(), tc.args)
			require.Error(t, err)
			if tc.name != "missing file" {
				assert.Equal(t, app.ErrorValidation, app.KindOf(err))
			}
			assert.Contains(t, err.Error(), tc.want)
			assert.Zero(t, calls)
		})
	}
}
func TestIssueCommentCommandsValidateLocallyAndRemoveDispatches(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no source", []string{"youtrack", "issue", "comment", "add", "APP-1"}, "exactly one of --text or --file is required"},
		{"both sources", []string{"youtrack", "issue", "comment", "add", "APP-1", "--text", "x", "--file", "x"}, "exactly one of --text or --file is required"},
		{"blank text", []string{"youtrack", "issue", "comment", "add", "APP-1", "--text", " \t"}, "comment text must not be blank"},
		{"missing file", []string{"youtrack", "issue", "comment", "add", "APP-1", "--file", "/missing/comment"}, "open /missing/comment"},
		{"wrong list arity", []string{"youtrack", "issue", "comment", "list"}, "usage: youtrack issue comment list <issue> [--limit <n>] [--offset <n>] [--all]"},
		{"blank removal ID", []string{"youtrack", "issue", "comment", "remove", "APP-1", " "}, "comment ID must not be blank"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
			defer server.Close()
			err := commentRoot(&bytes.Buffer{}, server).Run(context.Background(), tc.args)
			require.Error(t, err)
			if tc.name != "missing file" {
				assert.Equal(t, app.ErrorValidation, app.KindOf(err))
			}
			assert.Contains(t, err.Error(), tc.want)
			assert.Zero(t, calls)
		})
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"json", []string{"youtrack", "issue", "comment", "remove", "APP-1", "4-1", "--url", "SERVER", "--token", "secret", "--json"}, `"removed": true`},
		{"plain", []string{"youtrack", "issue", "comment", "remove", "APP-1", "4-1", "--url", "SERVER", "--token", "secret", "--plain"}, ""},
		{"human", []string{"youtrack", "issue", "comment", "remove", "APP-1", "4-1", "--url", "SERVER", "--token", "secret"}, "reversible"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/issues/APP-1/comments/4-1", r.URL.Path)
				var body map[string]bool
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, map[string]bool{"deleted": true}, body)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			out := &bytes.Buffer{}
			require.NoError(t, commentRoot(out, server).Run(context.Background(), replaceServer(tc.args, server.URL)))
			if tc.want == "" {
				assert.Empty(t, out.String())
			} else {
				assert.Contains(t, out.String(), tc.want)
			}
		})
	}
}

func replaceServer(args []string, server string) []string {
	out := append([]string(nil), args...)
	for i, arg := range out {
		if arg == "SERVER" {
			out[i] = server
		}
	}
	return out
}
