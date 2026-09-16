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
	"github.com/UsingCoding/youtrack-cli/internal/youtrack/dto"
)

func TestUpdateIssueSerializesResolvedFieldsAndTagsInOneRequest(t *testing.T) {
	var method, path, auth string
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, auth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL + "/youtrack", Token: "secret", HTTPClient: server.Client()})
	require.NoError(t, err)
	summary := "New"
	err = client.UpdateIssue(context.Background(), "TT-1", app.IssuePatch{
		Summary: &summary,
		Fields: []app.FieldAssignment{{
			Field: domain.FieldDefinition{ID: "pf-1", Name: "Priority", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle},
			Value: domain.EntityValue{ID: "priority-critical", Name: "Critical"},
		}},
		Tags: &[]domain.Tag{{ID: "tag-1", Name: "backend"}},
	})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, method)
	assert.Equal(t, "/youtrack/api/issues/TT-1", path)
	assert.Equal(t, "Bearer secret", auth)

	assert.Equal(t, "New", payload["summary"])
	fields, ok := payload["customFields"].([]any)
	require.True(t, ok)
	require.Len(t, fields, 1)
	field := fields[0].(map[string]any)
	assert.Equal(t, "SingleEnumIssueCustomField", field["$type"])
	assert.Equal(t, "pf-1", field["id"])
	assert.Equal(t, "priority-critical", field["value"].(map[string]any)["id"])
	tags := payload["tags"].([]any)
	assert.Equal(t, "tag-1", tags[0].(map[string]any)["id"])
}

func TestCreateIssueSendsOneNameBasedRequestAndMapsResponse(t *testing.T) {
	var calls int
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/issues", r.URL.Path)
		assert.Equal(t, issueFields, r.URL.Query().Get("fields"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"TT-1","summary":"New","description":"first\nsecond","created":1,"updated":2,"project":{"id":"0-1","name":"Tools","shortName":"TT"},"tags":[{"id":"tag-1","name":"backend"}],"customFields":[{"id":"pf-1","name":"Priority","$type":"SingleEnumIssueCustomField","value":{"id":"priority-critical","name":"Critical"}}]}`))
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	description := "first\nsecond"
	issue, err := client.CreateIssue(context.Background(), app.IssueCreate{
		Project: domain.Project{ID: "0-1"},
		Summary: "New", Description: &description,
		Fields: []app.FieldAssignment{{
			Field: domain.FieldDefinition{Name: "Priority", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle},
			Value: domain.EntityValue{ID: "priority-critical", Name: "Critical"},
		}},
		Tags: []domain.Tag{{ID: "tag-1"}},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, "0-1", payload["project"].(map[string]any)["id"])
	assert.Equal(t, "New", payload["summary"])
	assert.Equal(t, description, payload["description"])
	fields := payload["customFields"].([]any)
	field := fields[0].(map[string]any)
	assert.Equal(t, "Priority", field["name"])
	assert.NotContains(t, field, "id")
	assert.Equal(t, "SingleEnumIssueCustomField", field["$type"])
	assert.Equal(t, "priority-critical", field["value"].(map[string]any)["id"])
	assert.Equal(t, "tag-1", payload["tags"].([]any)[0].(map[string]any)["id"])
	assert.Equal(t, "TT-1", issue.IDReadable)
	assert.Equal(t, domain.EntityValue{ID: "priority-critical", Name: "Critical"}, issue.Fields[0].Value)
}

func TestCreateIssueDoesNotRetryFailures(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, `{"error":"unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	_, err = client.CreateIssue(context.Background(), app.IssueCreate{Project: domain.Project{ID: "0-1"}, Summary: "New"})

	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestGetIssueKeepsUnknownCustomFieldReadable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/issues/TT-1/sprints" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"TT-1","summary":"S","description":"","created":1,"updated":2,"project":{"id":"0-1","name":"Tools","shortName":"TT"},"tags":[],"customFields":[{"id":"x","name":"Future","$type":"FutureIssueCustomField","value":{"x":1}}]}`))
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	issue, err := client.GetIssue(context.Background(), "TT-1")
	require.NoError(t, err)
	require.Len(t, issue.Fields, 2)
	assert.Equal(t, domain.UnknownValue{Type: "FutureIssueCustomField"}, issue.Fields[1].Value)
}

func TestSerializeStateMachineTransitionUsesEvent(t *testing.T) {
	payload, err := serializeAssignment(app.FieldAssignment{
		Field: domain.FieldDefinition{ID: "82-11", Name: "State", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle},
		Value: domain.StateTransitionValue{ID: "in-progress", Presentation: "In Progress"},
	})
	require.NoError(t, err)
	assert.Equal(t, "StateMachineIssueCustomField", payload["$type"])
	assert.NotContains(t, payload, "value")
	event := payload["event"].(map[string]any)
	assert.Equal(t, "in-progress", event["id"])
	assert.Equal(t, "Event", event["$type"])
}

func TestRawAPIPreservesEndpointQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/issues", r.URL.Path)
		assert.Equal(t, "project: TT", r.URL.Query().Get("query"))
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	_, err = client.DoRaw(context.Background(), http.MethodGet, "/api/issues?query=project%3A+TT", nil, nil)
	require.NoError(t, err)
}

func TestMapStateMachineFieldKeepsCurrentStateAndTransitions(t *testing.T) {
	field, err := mapIssueField(dto.IssueCustomField{
		ID: "82-11", Name: "State", Type: "StateMachineIssueCustomField",
		Value:          json.RawMessage(`{"id":"57-19","name":"Open"}`),
		PossibleEvents: []dto.Event{{ID: "in-progress", Presentation: "In Progress"}},
	})

	require.NoError(t, err)
	assert.True(t, field.StateMachine)
	assert.Equal(t, domain.EntityValue{ID: "57-19", Name: "Open"}, field.Value)
	assert.Equal(t, []domain.FieldOption{{ID: "in-progress", Name: "In Progress"}}, field.Transitions)
}

func TestSerializeDateUsesMidnightTimestampAsProvided(t *testing.T) {
	day := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	value, err := serializeValue(domain.FieldDefinition{Kind: domain.FieldDate, Cardinality: domain.CardinalitySingle}, domain.DateValue{Value: day})
	require.NoError(t, err)
	assert.Equal(t, day.UnixMilli(), value)
}
