package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack"
)

type editHarness struct {
	t        *testing.T
	server   *httptest.Server
	calls    []string
	posts    int
	commands int
	board    bool
	payload  map[string]any
}

func newEditHarness(t *testing.T) *editHarness {
	t.Helper()
	h := &editHarness{t: t}
	h.server = httptest.NewServer(http.HandlerFunc(h.serveHTTP))
	t.Cleanup(h.server.Close)
	return h
}

func (h *editHarness) serveHTTP(w http.ResponseWriter, r *http.Request) {
	h.calls = append(h.calls, r.Method+" "+r.URL.Path)

	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/issues/TT-1":
		if h.posts > 0 {
			h.writeJSON(w, issueResponse("Changed", "p-critical", "Critical"))
			return
		}
		h.writeJSON(w, issueResponse("Old", "p-normal", "Normal"))
	case r.Method == http.MethodGet && r.URL.Path == "/api/issues/TT-1/sprints":
		if h.board && queryInt(h.t, r, "$skip") == 0 {
			h.writeJSON(w, []any{map[string]any{"agile": map[string]any{"id": "a-platform", "name": "Platform Board"}}})
			return
		}
		h.writeJSON(w, []any{})
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/projects/0-1/customFields":
		skip := queryInt(h.t, r, "$skip")
		if skip == 0 {
			h.writeJSON(w, []any{map[string]any{
				"id": "pf-priority", "canBeEmpty": true,
				"field": map[string]any{
					"id": "f-priority", "name": "Priority",
					"fieldType": map[string]any{"id": "enum[1]", "valueType": "enum", "isMultiValue": false},
				},
				"bundle": map[string]any{"id": "b-priority"},
			}})
			return
		}
		h.writeJSON(w, []any{})
	case r.Method == http.MethodGet && r.URL.Path == "/api/admin/projects/0-1/customFields/pf-priority/bundle/values":
		skip := queryInt(h.t, r, "$skip")
		if skip == 0 {
			h.writeJSON(w, []any{map[string]any{"id": "p-critical", "name": "Critical", "archived": false}})
			return
		}
		h.writeJSON(w, []any{})
	case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
		require.Equal(h.t, "backend", r.URL.Query().Get("query"))
		skip := queryInt(h.t, r, "$skip")
		if skip == 0 {
			h.writeJSON(w, []any{map[string]any{"id": "tag-backend", "name": "backend"}})
			return
		}
		h.writeJSON(w, []any{})
	case r.Method == http.MethodGet && r.URL.Path == "/api/agiles":
		if queryInt(h.t, r, "$skip") == 0 {
			h.writeJSON(w, []any{map[string]any{"id": "a-platform", "name": "Platform Board", "projects": []any{map[string]any{"id": "0-1"}}}})
			return
		}
		h.writeJSON(w, []any{})
	case r.Method == http.MethodPost && r.URL.Path == "/api/commands/assist":
		h.writeJSON(w, map[string]any{"commands": []any{map[string]any{"error": false, "delete": false, "description": "ok"}}})
	case r.Method == http.MethodPost && r.URL.Path == "/api/commands":
		h.commands++
		h.board = true
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodPost && r.URL.Path == "/api/issues/TT-1":
		h.posts++
		require.NoError(h.t, json.NewDecoder(r.Body).Decode(&h.payload))
		h.writeJSON(w, issueResponse("Changed", "p-critical", "Critical"))
	default:
		h.t.Errorf("unexpected request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		http.Error(w, `{"error":"unexpected request"}`, http.StatusNotFound)
	}
}

func issueResponse(summary, priorityID, priorityName string) map[string]any {
	return map[string]any{
		"id": "2-1", "idReadable": "TT-1", "summary": summary, "description": "",
		"created": int64(1_700_000_000_000), "updated": int64(1_700_000_000_000),
		"project": map[string]any{"id": "0-1", "name": "Tools", "shortName": "TT"},
		"tags":    []any{map[string]any{"id": "tag-legacy", "name": "legacy"}},
		"customFields": []any{map[string]any{
			"id": "pf-priority", "name": "Priority", "$type": "SingleEnumIssueCustomField",
			"value": map[string]any{"id": priorityID, "name": priorityName},
		}},
	}
}

func (h *editHarness) writeJSON(w http.ResponseWriter, value any) {
	require.NoError(h.t, json.NewEncoder(w).Encode(value))
}

func queryInt(t *testing.T, r *http.Request, name string) int {
	t.Helper()
	v, err := strconv.Atoi(r.URL.Query().Get(name))
	require.NoError(t, err)
	return v
}

func (h *editHarness) service(t *testing.T) *app.Service {
	t.Helper()
	client, err := youtrack.NewClient(youtrack.Options{BaseURL: h.server.URL, HTTPClient: h.server.Client(), Token: "secret"})
	require.NoError(t, err)
	return app.NewService(client, client, client, client, client, client, client, client, client, client)
}

func TestCombinedIssueEditResolvesThenUsesOneMutation(t *testing.T) {
	h := newEditHarness(t)
	service := h.service(t)
	summary := "Changed"

	got, err := service.EditIssue(context.Background(), "TT-1", app.EditRequest{
		Summary: &summary,
		Fields:  []app.FieldInput{{Name: "Priority", Value: "Critical"}},
		AddTags: []string{"backend"}, RemoveTags: []string{"legacy"},
	})

	require.NoError(t, err)
	assert.Equal(t, "Changed", got.Summary)
	assert.Equal(t, 1, h.posts)
	require.NotEmpty(t, h.calls)
	assert.Contains(t, h.calls, "POST /api/issues/TT-1", "all resolution reads must finish before mutation")
	assert.Equal(t, "Changed", h.payload["summary"])

	fields, ok := h.payload["customFields"].([]any)
	require.True(t, ok)
	require.Len(t, fields, 1)
	field := fields[0].(map[string]any)
	assert.Equal(t, "SingleEnumIssueCustomField", field["$type"])
	value := field["value"].(map[string]any)
	assert.Equal(t, "p-critical", value["id"])

	tags, ok := h.payload["tags"].([]any)
	require.True(t, ok)
	require.Len(t, tags, 1)
	assert.Equal(t, "tag-backend", tags[0].(map[string]any)["id"])
}

func TestCombinedIssueEditInvalidLastFieldMakesZeroMutations(t *testing.T) {
	h := newEditHarness(t)
	service := h.service(t)
	summary := "Changed"

	_, err := service.EditIssue(context.Background(), "TT-1", app.EditRequest{
		Summary: &summary,
		Fields:  []app.FieldInput{{Name: "Priority", Value: "Does not exist"}},
	})

	require.Error(t, err)
	assert.Equal(t, 0, h.posts)
	for _, call := range h.calls {
		assert.NotEqual(t, "POST /api/issues/TT-1", call)
	}
}

func TestCombinedIssueEditBoardValidatesThenMutatesInOrder(t *testing.T) {
	h := newEditHarness(t)
	service := h.service(t)
	summary := "Changed"

	got, err := service.EditIssue(context.Background(), "TT-1", app.EditRequest{
		Summary: &summary, Fields: []app.FieldInput{{Name: "Priority", Value: "Critical"}, {Name: "Board", Value: "Platform Board"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "Changed", got.Summary)
	assert.Equal(t, 1, h.posts)
	assert.Equal(t, 1, h.commands)
	require.NotEmpty(t, got.Fields)
	assert.Equal(t, "Board", got.Fields[0].Name)

	assist, issuePost, command := -1, -1, -1
	for i, call := range h.calls {
		switch call {
		case "POST /api/commands/assist":
			assist = i
		case "POST /api/issues/TT-1":
			issuePost = i
		case "POST /api/commands":
			command = i
		}
	}
	assert.GreaterOrEqual(t, assist, 0)
	assert.Greater(t, issuePost, assist)
	assert.Greater(t, command, issuePost)
}
