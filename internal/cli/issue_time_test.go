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

func timeCLIItem() map[string]any {
	return map[string]any{"id": "115-7", "date": int64(1790640000000), "duration": map[string]any{"minutes": 90}, "text": "note", "author": nil, "creator": nil, "type": nil, "created": int64(1790683200000), "updated": nil}
}

func TestIssueTimeCommandsDispatchAndPreserveText(t *testing.T) {
	var postBodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/issues/APP-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "2-1", "idReadable": "APP-1", "project": map[string]any{"id": "0-1"}, "tags": []any{}, "customFields": []any{}})
		case "GET /api/issues/APP-1/sprints":
			_, _ = w.Write([]byte(`[]`))
		case "GET /api/admin/projects/0-1/timeTrackingSettings/workItemTypes":
			if r.URL.Query().Get("$skip") == "0" {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "development", "name": "Development"}})
			} else {
				_, _ = w.Write([]byte(`[]`))
			}
		case "GET /api/issues/2-1/timeTracking/workItems":
			_ = json.NewEncoder(w).Encode([]map[string]any{timeCLIItem()})
		case "GET /api/issues/2-1/timeTracking/workItems/115-7":
			_ = json.NewEncoder(w).Encode(timeCLIItem())
		case "GET /api/admin/projects/0-1/timeTrackingSettings":
			_, _ = w.Write([]byte(`{"enabled":true}`))
		case "POST /api/issues/2-1/timeTracking/workItems", "POST /api/issues/2-1/timeTracking/workItems/115-7":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			postBodies = append(postBodies, body)
			_ = json.NewEncoder(w).Encode(timeCLIItem())
		case "DELETE /api/issues/2-1/timeTracking/workItems/115-7":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	for _, args := range [][]string{
		{"youtrack", "issue", "time", "types", "APP-1", "--all", "--url", "SERVER", "--token", "secret", "--plain"},
		{"youtrack", "issue", "time", "list", "APP-1", "--limit", "1", "--url", "SERVER", "--token", "secret", "--plain"},
		{"youtrack", "issue", "time", "view", "APP-1", "115-7", "--url", "SERVER", "--token", "secret", "--json"},
		{"youtrack", "issue", "time", "add", "APP-1", "--duration", "1h30m", "--date", "2026-09-29", "--text", "", "--url", "SERVER", "--token", "secret", "--json"},
		{"youtrack", "issue", "time", "edit", "APP-1", "115-7", "--text", "", "--clear-type", "--url", "SERVER", "--token", "secret", "--json"},
		{"youtrack", "issue", "time", "remove", "APP-1", "115-7", "--yes", "--url", "SERVER", "--token", "secret", "--plain"},
	} {
		out := &bytes.Buffer{}
		require.NoError(t, issueSearchRoot(out, server).Run(context.Background(), replaceServer(args, server.URL)))
	}
	require.Len(t, postBodies, 2)
	assert.Equal(t, "", postBodies[0]["text"])
	assert.Equal(t, map[string]any{"text": "", "type": nil}, postBodies[1])
}

func TestIssueTimeLocalValidationPreventsHTTP(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	cases := [][]string{
		{"youtrack", "issue", "time", "add", "APP-1", "--url", "SERVER", "--token", "secret"},
		{"youtrack", "issue", "time", "add", "APP-1", "--duration", "1m", "--date", "2026-09-29", "--text", "x", "--file", "x", "--url", "SERVER", "--token", "secret"},
		{"youtrack", "issue", "time", "edit", "APP-1", "115-7", "--type", "x", "--clear-type", "--url", "SERVER", "--token", "secret"},
		{"youtrack", "issue", "time", "edit", "APP-1", "115-7", "--url", "SERVER", "--token", "secret"},
		{"youtrack", "issue", "time", "remove", "APP-1", "115-7", "--url", "SERVER", "--token", "secret"},
	}
	for _, args := range cases {
		err := issueSearchRoot(&bytes.Buffer{}, server).Run(context.Background(), replaceServer(args, server.URL))
		require.Error(t, err)
	}
	assert.Zero(t, calls)
}

