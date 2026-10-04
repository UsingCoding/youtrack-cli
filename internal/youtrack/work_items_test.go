package youtrack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func TestWorkItemAdapterUsesScopedPathsAndPartialPayloads(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			assert.Equal(t, workItemFields, r.URL.Query().Get("fields"))
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /api/issues/2-1/timeTracking/workItems":
			assert.Equal(t, "3", r.URL.Query().Get("$skip"))
			assert.Equal(t, "2", r.URL.Query().Get("$top"))
			_, _ = w.Write([]byte(`[{"id":"115-7","date":1790640000000,"duration":{"minutes":90},"text":null,"author":null,"creator":null,"type":null,"created":1790683200000,"updated":null}]`))
		case "GET /api/issues/2-1/timeTracking/workItems/115-7":
			_, _ = w.Write([]byte(`{"id":"115-7","date":1790640000000,"duration":{"minutes":90},"text":"note","author":{"id":"u","login":"alice","fullName":"Alice"},"creator":null,"type":null,"created":1790683200000,"updated":null}`))
		case "POST /api/issues/2-1/timeTracking/workItems", "POST /api/issues/2-1/timeTracking/workItems/115-7":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			bodies = append(bodies, body)
			_, _ = w.Write([]byte(`{"id":"115-7","date":1790640000000,"duration":{"minutes":90},"text":null,"author":null,"creator":null,"type":null,"created":1790683200000,"updated":null}`))
		case "DELETE /api/issues/2-1/timeTracking/workItems/115-7":
			assert.Equal(t, int64(0), r.ContentLength)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	items, err := client.ListWorkItems(context.Background(), "2-1", app.Page{Offset: 3, Limit: 2})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), items[0].Date)
	assert.Nil(t, items[0].Text)
	got, err := client.GetWorkItem(context.Background(), "2-1", "115-7")
	require.NoError(t, err)
	assert.Equal(t, "alice", got.Author.Login)

	date := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	text := ""
	_, err = client.CreateWorkItem(context.Background(), "2-1", app.WorkItemCreate{Date: date, DurationMinutes: 90, Text: &text, Type: &domain.WorkItemType{ID: "development"}, Author: &domain.User{ID: "alice"}})
	require.NoError(t, err)
	_, err = client.UpdateWorkItem(context.Background(), "2-1", "115-7", app.WorkItemPatch{Text: &text, Type: app.WorkItemTypePatch{Set: true}})
	require.NoError(t, err)
	require.NoError(t, client.DeleteWorkItem(context.Background(), "2-1", "115-7"))
	require.Len(t, bodies, 2)
	assert.Equal(t, float64(1790640000000), bodies[0]["date"])
	assert.Equal(t, float64(90), bodies[0]["duration"].(map[string]any)["minutes"])
	assert.Equal(t, "", bodies[0]["text"])
	assert.Equal(t, map[string]any{"id": "development"}, bodies[0]["type"])
	assert.Equal(t, map[string]any{"id": "alice"}, bodies[0]["author"])
	assert.Equal(t, map[string]any{"text": "", "type": nil}, bodies[1])
}

func TestWorkItemAdapterSettingsTypesAndMutationFailureAreSingleAttempt(t *testing.T) {
	postCalls, deleteCalls := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/admin/projects/0-1/timeTrackingSettings":
			assert.Equal(t, "enabled", r.URL.Query().Get("fields"))
			_, _ = w.Write([]byte(`{"enabled":true}`))
		case "/api/admin/projects/0-1/timeTrackingSettings/workItemTypes":
			assert.Equal(t, workItemTypeFields, r.URL.Query().Get("fields"))
			assert.Equal(t, "2", r.URL.Query().Get("$skip"))
			_, _ = w.Write([]byte(`[{"id":"t","name":"Development"}]`))
		case "/api/issues/2-1/timeTracking/workItems":
			postCalls++
			http.Error(w, `{"error":"no"}`, http.StatusServiceUnavailable)
		case "/api/issues/2-1/timeTracking/workItems/115-7":
			deleteCalls++
			http.Error(w, `{"error":"no"}`, http.StatusServiceUnavailable)
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	enabled, err := client.TimeTrackingEnabled(context.Background(), "0-1")
	require.NoError(t, err)
	assert.True(t, enabled)
	types, err := client.ListWorkItemTypes(context.Background(), "0-1", app.Page{Offset: 2, Limit: 1})
	require.NoError(t, err)
	assert.Equal(t, []domain.WorkItemType{{ID: "t", Name: "Development"}}, types)
	_, err = client.CreateWorkItem(context.Background(), "2-1", app.WorkItemCreate{Date: time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), DurationMinutes: 1})
	require.Error(t, err)
	require.Error(t, client.DeleteWorkItem(context.Background(), "2-1", "115-7"))
	assert.Equal(t, 1, postCalls)
	assert.Equal(t, 1, deleteCalls)
}
