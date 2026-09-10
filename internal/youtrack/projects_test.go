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
