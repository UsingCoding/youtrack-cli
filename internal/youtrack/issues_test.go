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

func TestMoveIssueAndTagEndpointsUseExpectedWireContracts(t *testing.T) {
	var requests []struct {
		Method string
		Path   string
		Body   map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		item := struct {
			Method string
			Path   string
			Body   map[string]any
		}{Method: r.Method, Path: r.URL.Path}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&item.Body)
		}
		requests = append(requests, item)
		switch r.URL.Path {
		case "/api/issues/TT-1/project", "/api/issues/TT-1/tags", "/api/issues/TT-1/tags/tag-1":
			w.WriteHeader(http.StatusNoContent)
		case "/api/issues/TT-1":
			_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"TT-1","summary":"Moved","created":1,"updated":2,"project":{"id":"0-2","name":"Platform","shortName":"PF"}}`))
		case "/api/issues/TT-1/sprints":
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	moved, err := client.MoveIssue(context.Background(), "TT-1", "0-2")
	require.NoError(t, err)
	require.NoError(t, client.AddIssueTag(context.Background(), "TT-1", domain.Tag{ID: "tag-1"}))
	require.NoError(t, client.RemoveIssueTag(context.Background(), "TT-1", domain.Tag{ID: "tag-1"}))

	assert.Equal(t, "0-2", moved.Project.ID)
	require.Len(t, requests, 5)
	assert.Equal(t, http.MethodPost, requests[0].Method)
	assert.Equal(t, "/api/issues/TT-1/project", requests[0].Path)
	assert.Equal(t, "0-2", requests[0].Body["id"])
	assert.Equal(t, http.MethodPost, requests[3].Method)
	assert.Equal(t, "/api/issues/TT-1/tags", requests[3].Path)
	assert.Equal(t, "tag-1", requests[3].Body["id"])
	assert.Equal(t, http.MethodDelete, requests[4].Method)
	assert.Equal(t, "/api/issues/TT-1/tags/tag-1", requests[4].Path)
}

func TestMapIssueFieldMapsReadableValueShapes(t *testing.T) {
	tests := []struct {
		name  string
		field dto.IssueCustomField
		want  domain.FieldValue
	}{
		{name: "null", field: dto.IssueCustomField{Type: "SingleEnumIssueCustomField", Value: json.RawMessage(`null`)}, want: domain.EmptyValue{}},
		{name: "entity", field: dto.IssueCustomField{Type: "SingleEnumIssueCustomField", Value: json.RawMessage(`{"id":"e","name":"Entity"}`)}, want: domain.EntityValue{ID: "e", Name: "Entity"}},
		{name: "multi entity", field: dto.IssueCustomField{Type: "MultiEnumIssueCustomField", Value: json.RawMessage(`[{"id":"e","name":"Entity"}]`)}, want: domain.MultiValue{Values: []domain.FieldValue{domain.EntityValue{ID: "e", Name: "Entity"}}}},
		{name: "user", field: dto.IssueCustomField{Type: "SingleUserIssueCustomField", Value: json.RawMessage(`{"id":"u","login":"alice","fullName":"Alice"}`)}, want: domain.UserValue{ID: "u", Login: "alice", FullName: "Alice"}},
		{name: "multi user", field: dto.IssueCustomField{Type: "MultiUserIssueCustomField", Value: json.RawMessage(`[{"id":"u","login":"alice","fullName":"Alice"}]`)}, want: domain.MultiValue{Values: []domain.FieldValue{domain.UserValue{ID: "u", Login: "alice", FullName: "Alice"}}}},
		{name: "string", field: dto.IssueCustomField{Type: "SimpleIssueCustomField", Value: json.RawMessage(`"text"`)}, want: domain.StringValue{Value: "text"}},
		{name: "integer", field: dto.IssueCustomField{Type: "SimpleIssueCustomField", Value: json.RawMessage(`5`)}, want: domain.IntegerValue{Value: 5}},
		{name: "float", field: dto.IssueCustomField{Type: "SimpleIssueCustomField", Value: json.RawMessage(`5.5`)}, want: domain.FloatValue{Value: 5.5}},
		{name: "date", field: dto.IssueCustomField{Type: "DateIssueCustomField", Value: json.RawMessage(`1000`)}, want: domain.DateValue{Value: time.UnixMilli(1000)}},
		{name: "period", field: dto.IssueCustomField{Type: "PeriodIssueCustomField", Value: json.RawMessage(`{"minutes":90}`)}, want: domain.PeriodValue{Minutes: 90}},
		{name: "text", field: dto.IssueCustomField{Type: "TextIssueCustomField", Value: json.RawMessage(`{"text":"long"}`)}, want: domain.TextValue{Value: "long"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapIssueField(tt.field)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Value)
		})
	}
	t.Run("malformed", func(t *testing.T) {
		_, err := mapIssueField(dto.IssueCustomField{Type: "SingleUserIssueCustomField", Value: json.RawMessage(`{`)})
		require.Error(t, err)
	})
}

func TestSerializeAssignmentsCoversSupportedKindsAndValues(t *testing.T) {
	tests := []struct {
		name     string
		def      domain.FieldDefinition
		value    domain.FieldValue
		wantType string
		want     any
	}{
		{name: "enum", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle}, value: domain.EntityValue{ID: "e"}, wantType: "SingleEnumIssueCustomField", want: map[string]any{"id": "e"}},
		{name: "multi enum", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldEnum, Cardinality: domain.CardinalityMulti}, value: domain.MultiValue{Values: []domain.FieldValue{domain.EntityValue{ID: "e"}}}, wantType: "MultiEnumIssueCustomField", want: []any{map[string]any{"id": "e"}}},
		{name: "state", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle}, value: domain.EntityValue{ID: "s"}, wantType: "StateIssueCustomField", want: map[string]any{"id": "s"}},
		{name: "user", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldUser, Cardinality: domain.CardinalitySingle}, value: domain.UserValue{ID: "u"}, wantType: "SingleUserIssueCustomField", want: map[string]any{"id": "u"}},
		{name: "multi user", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldUser, Cardinality: domain.CardinalityMulti}, value: domain.MultiValue{Values: []domain.FieldValue{domain.UserValue{ID: "u"}}}, wantType: "MultiUserIssueCustomField", want: []any{map[string]any{"id": "u"}}},
		{name: "version", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldVersion, Cardinality: domain.CardinalitySingle}, value: domain.EntityValue{ID: "v"}, wantType: "SingleVersionIssueCustomField", want: map[string]any{"id": "v"}},
		{name: "build", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldBuild, Cardinality: domain.CardinalityMulti}, value: domain.EmptyValue{}, wantType: "MultiBuildIssueCustomField", want: []any{}},
		{name: "owned", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldOwned, Cardinality: domain.CardinalitySingle}, value: domain.EntityValue{ID: "o"}, wantType: "SingleOwnedIssueCustomField", want: map[string]any{"id": "o"}},
		{name: "group", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldGroup, Cardinality: domain.CardinalityMulti}, value: domain.MultiValue{Values: []domain.FieldValue{domain.EntityValue{ID: "g"}}}, wantType: "MultiGroupIssueCustomField", want: []any{map[string]any{"id": "g"}}},
		{name: "date", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldDate, Cardinality: domain.CardinalitySingle}, value: domain.DateValue{Value: time.UnixMilli(1000)}, wantType: "DateIssueCustomField", want: int64(1000)},
		{name: "period", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldPeriod, Cardinality: domain.CardinalitySingle}, value: domain.PeriodValue{Minutes: 5}, wantType: "PeriodIssueCustomField", want: map[string]any{"minutes": int64(5)}},
		{name: "text", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldText, Cardinality: domain.CardinalitySingle}, value: domain.TextValue{Value: "text"}, wantType: "TextIssueCustomField", want: map[string]any{"text": "text"}},
		{name: "integer", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldInteger, Cardinality: domain.CardinalitySingle}, value: domain.IntegerValue{Value: 2}, wantType: "SimpleIssueCustomField", want: int64(2)},
		{name: "float", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldFloat, Cardinality: domain.CardinalitySingle}, value: domain.FloatValue{Value: 2.5}, wantType: "SimpleIssueCustomField", want: 2.5},
		{name: "empty scalar", def: domain.FieldDefinition{ID: "x", Kind: domain.FieldString, Cardinality: domain.CardinalitySingle}, value: domain.EmptyValue{}, wantType: "SimpleIssueCustomField", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := serializeAssignment(app.FieldAssignment{Field: tt.def, Value: tt.value})
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, payload["$type"])
			assert.Equal(t, tt.want, payload["value"])
		})
	}
	t.Run("transition cannot create", func(t *testing.T) {
		_, err := serializeCreateAssignment(app.FieldAssignment{Field: domain.FieldDefinition{Name: "State", Kind: domain.FieldState}, Value: domain.StateTransitionValue{ID: "go"}})
		require.Error(t, err)
		assert.Equal(t, app.ErrorValidation, app.KindOf(err))
	})
	t.Run("unsupported field and value", func(t *testing.T) {
		_, err := serializeAssignment(app.FieldAssignment{Field: domain.FieldDefinition{Kind: domain.FieldUnknown}, Value: domain.StringValue{Value: "x"}})
		require.Error(t, err)
		_, err = serializeAssignment(app.FieldAssignment{Field: domain.FieldDefinition{Name: "Field", Kind: domain.FieldString}, Value: domain.UnknownValue{Type: "future"}})
		require.Error(t, err)
	})
}
