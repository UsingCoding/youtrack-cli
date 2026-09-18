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

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func TestIssueTagCommands(t *testing.T) {
	postCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/issues/TT-1":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"TT-1","summary":"Issue","created":1,"updated":2,"project":{"id":"0-1","name":"Tools","shortName":"TT"},"tags":[]}`))
		case "/api/issues/TT-1/sprints":
			_, _ = w.Write([]byte(`[]`))
		case "/api/admin/projects/0-1/customFields":
			_, _ = w.Write([]byte(`[]`))
		case "/api/tags":
			require.Equal(t, http.MethodGet, r.Method)
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"tag-backend","name":"backend"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case "/api/issues/TT-1/tags":
			postCalls++
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			assert.Equal(t, "tag-backend", payload["id"])
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	out := &bytes.Buffer{}
	err := issueSearchRoot(out, server).Run(context.Background(), []string{"youtrack", "issue", "tag", "add", "TT-1", "backend", "--url", server.URL, "--token", "secret", "--json"})
	require.NoError(t, err)
	assert.Equal(t, 1, postCalls)
}

func TestIssueFieldSetRejectsInvalidArgumentsBeforeHTTP(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	err := issueSearchRoot(&bytes.Buffer{}, server).Run(context.Background(), []string{"youtrack", "issue", "field", "set", "TT-1", "Priority"})
	require.Error(t, err)
	assert.Equal(t, app.ErrorValidation, app.KindOf(err))
	assert.Zero(t, calls)
}
