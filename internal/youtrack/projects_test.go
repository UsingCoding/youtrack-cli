package youtrack

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

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func TestListProjectFieldsPaginatesPastServerDefaultLimit(t *testing.T) {
	var skips []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/admin/projects/0-1/customFields", r.URL.Path)
		skip, err := strconv.Atoi(r.URL.Query().Get("$skip"))
		require.NoError(t, err)
		skips = append(skips, skip)
		count := 0
		switch skip {
		case 0:
			count = 42
		case 42:
			count = 1
		}
		page := make([]map[string]any, 0, count)
		for i := 0; i < count; i++ {
			id := skip + i
			page = append(page, map[string]any{
				"id": "pf-" + strconv.Itoa(id), "canBeEmpty": true,
				"field": map[string]any{"id": "f-" + strconv.Itoa(id), "name": "Field " + strconv.Itoa(id), "fieldType": map[string]any{"id": "string", "valueType": "string"}},
			})
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(page))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	fields, err := client.ListProjectFields(context.Background(), "0-1")
	require.NoError(t, err)
	assert.Len(t, fields, 43)
	assert.Equal(t, []int{0, 42, 43}, skips)
	assert.Equal(t, domain.FieldString, fields[0].Kind)
}

func TestParseFieldTypeSeparatesSemanticKindAndCardinality(t *testing.T) {
	kind, cardinality, err := parseFieldType("user[*]", "user", false)
	require.NoError(t, err)
	assert.Equal(t, domain.FieldUser, kind)
	assert.Equal(t, domain.CardinalityMulti, cardinality)

	kind, cardinality, err = parseFieldType("date and time", "date and time", false)
	require.NoError(t, err)
	assert.Equal(t, domain.FieldDateTime, kind)
	assert.Equal(t, domain.CardinalitySingle, cardinality)

	kind, cardinality, err = parseFieldType("user", "user", true)
	require.NoError(t, err)
	assert.Equal(t, domain.FieldUser, kind)
	assert.Equal(t, domain.CardinalityMulti, cardinality)
}

func TestProjectsAndFieldBundlesMapPaginatedResponses(t *testing.T) {
	var projectSkips, optionSkips, userSkips []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/admin/projects/0-1":
			require.Equal(t, projectFields, r.URL.Query().Get("fields"))
			_, _ = w.Write([]byte(`{"id":"0-1","name":"Tools","shortName":"TT"}`))
		case "/api/admin/projects":
			projectSkips = append(projectSkips, r.URL.Query().Get("$skip"))
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"0-1","name":"Tools","shortName":"TT"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case "/api/admin/projects/0-1/customFields/pf-1/bundle/values":
			optionSkips = append(optionSkips, r.URL.Query().Get("$skip"))
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"one","name":"One"},{"id":"old","name":"Old","archived":true}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case "/api/admin/customFieldSettings/bundles/user/users/aggregatedUsers":
			userSkips = append(userSkips, r.URL.Query().Get("$skip"))
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"u-1","login":"alice","fullName":"Alice"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	project, err := client.GetProject(context.Background(), "0-1")
	require.NoError(t, err)
	projects, err := client.SearchProjects(context.Background(), "tool")
	require.NoError(t, err)
	options, err := client.ListFieldOptions(context.Background(), "0-1", domain.FieldDefinition{ID: "pf-1"})
	require.NoError(t, err)
	users, err := client.ListFieldUsers(context.Background(), "0-1", domain.FieldDefinition{ID: "pf-user", Name: "Assignee", BundleID: "users"})
	require.NoError(t, err)

	assert.Equal(t, domain.Project{ID: "0-1", Name: "Tools", ShortName: "TT"}, project)
	assert.Equal(t, []domain.Project{{ID: "0-1", Name: "Tools", ShortName: "TT"}}, projects)
	assert.Equal(t, []string{"0", "1"}, projectSkips)
	assert.Equal(t, []domain.FieldOption{{ID: "one", Name: "One"}}, options)
	assert.Equal(t, []string{"0", "2"}, optionSkips)
	assert.Equal(t, []domain.User{{ID: "u-1", Login: "alice", FullName: "Alice"}}, users)
	assert.Equal(t, []string{"0", "1"}, userSkips)
}

func TestListFieldUsersRequiresConfiguredBundle(t *testing.T) {
	client, err := NewClient(Options{BaseURL: "https://youtrack.example"})
	require.NoError(t, err)
	_, err = client.ListFieldUsers(context.Background(), "0-1", domain.FieldDefinition{Name: "Assignee"})
	require.Error(t, err)
	assert.Equal(t, app.ErrorRuntime, app.KindOf(err))
}

func TestParseFieldTypeMapsEverySemanticKind(t *testing.T) {
	tests := []struct {
		id   string
		kind domain.FieldKind
		card domain.Cardinality
	}{
		{id: "string", kind: domain.FieldString},
		{id: "integer", kind: domain.FieldInteger},
		{id: "float", kind: domain.FieldFloat},
		{id: "date", kind: domain.FieldDate},
		{id: "date and time", kind: domain.FieldDateTime},
		{id: "dateTime", kind: domain.FieldDateTime},
		{id: "period", kind: domain.FieldPeriod},
		{id: "text", kind: domain.FieldText},
		{id: "enum", kind: domain.FieldEnum},
		{id: "state", kind: domain.FieldState},
		{id: "user", kind: domain.FieldUser},
		{id: "version", kind: domain.FieldVersion},
		{id: "build", kind: domain.FieldBuild},
		{id: "ownedField", kind: domain.FieldOwned},
		{id: "owned", kind: domain.FieldOwned},
		{id: "group", kind: domain.FieldGroup},
		{id: "enum[*]", kind: domain.FieldEnum, card: domain.CardinalityMulti},
		{id: "enum[1]", kind: domain.FieldEnum},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			kind, card, err := parseFieldType(tt.id, "", false)
			require.NoError(t, err)
			assert.Equal(t, tt.kind, kind)
			if tt.card == "" {
				tt.card = domain.CardinalitySingle
			}
			assert.Equal(t, tt.card, card)
		})
	}
	kind, card, err := parseFieldType("", "unknown", true)
	require.Error(t, err)
	assert.Equal(t, domain.FieldUnknown, kind)
	assert.Equal(t, domain.CardinalityMulti, card)
}
