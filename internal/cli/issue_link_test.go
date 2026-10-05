package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueLinkCommandsDispatchWithNestedGlobals(t *testing.T) {
	present := false
	var postBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/issueLinkTypes":
			if r.URL.Query().Get("$skip") == "0" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "parent", "name": "Parent", "directed": true, "aggregation": true, "sourceToTarget": "parent for", "targetToSource": "subtask of"}})
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case "GET /api/issues/APP-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "2-1", "idReadable": "APP-1"})
		case "GET /api/issues/APP-2":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "2-2", "idReadable": "APP-2"})
		case "GET /api/issues/2-1/links/parentt/issues":
			if present && r.URL.Query().Get("$skip") == "0" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "2-2", "idReadable": "APP-2", "summary": "Parent", "project": map[string]any{"id": "0-1", "name": "App", "shortName": "APP"}, "created": 0, "updated": 0}})
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case "POST /api/issues/2-1/links/parentt/issues":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&postBody))
			present = true
			w.WriteHeader(http.StatusNoContent)
		case "DELETE /api/issues/2-1/links/parentt/issues/2-2":
			present = false
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	commands := [][]string{
		{"youtrack", "issue", "link", "types", "--all", "--url", "SERVER", "--token", "secret", "--plain"},
		{"youtrack", "issue", "link", "list", "APP-1", "--type", "parent", "--direction", "inward", "--limit", "1", "--url", "SERVER", "--token", "secret", "--plain"},
		{"youtrack", "issue", "link", "add", "APP-1", "APP-2", "--type", "parent", "--direction", "inward", "--url", "SERVER", "--token", "secret", "--json"},
		{"youtrack", "issue", "link", "remove", "APP-1", "APP-2", "--type", "parent", "--direction", "inward", "--url", "SERVER", "--token", "secret", "--json"},
	}
	for _, args := range commands {
		out := &bytes.Buffer{}
		require.NoError(t, issueSearchRoot(out, server).Run(context.Background(), replaceServer(args, server.URL)))
	}
	assert.Equal(t, map[string]any{"id": "2-2"}, postBody)
	assert.False(t, present)
}

func TestIssueLinkLocalValidationPreventsHTTP(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	cases := [][]string{
		{"youtrack", "issue", "link", "list", "APP-1", "--url", "SERVER", "--token", "secret"},
		{"youtrack", "issue", "link", "list", "APP-1", "--type", " ", "--url", "SERVER", "--token", "secret"},
		{"youtrack", "issue", "link", "list", "APP-1", "--type", "parent", "--direction", "sideways", "--url", "SERVER", "--token", "secret"},
		{"youtrack", "issue", "link", "add", "APP-1", "APP-2", "--type", "parent", "--direction", "", "--url", "SERVER", "--token", "secret"},
	}
	for _, args := range cases {
		err := issueSearchRoot(&bytes.Buffer{}, server).Run(context.Background(), replaceServer(args, server.URL))
		require.Error(t, err)
	}
	assert.Zero(t, calls)
}
