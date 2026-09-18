package output

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func TestJSONUsesStableSemanticFieldShape(t *testing.T) {
	var out bytes.Buffer
	renderer, err := New(&out, true, false)
	require.NoError(t, err)
	issue := domain.Issue{
		ID: "2-1", IDReadable: "TT-1", Summary: "Hello", Project: domain.Project{ID: "0-1", Name: "Tools", ShortName: "TT"},
		Fields: []domain.IssueField{{ID: "pf-1", Name: "Priority", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle, Value: domain.EntityValue{ID: "p-1", Name: "Critical"}}},
	}
	require.NoError(t, renderer.Issue(issue))

	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, "TT-1", got["id"])
	fields := got["fields"].([]any)
	field := fields[0].(map[string]any)
	assert.Equal(t, "enum", field["type"])
	assert.NotContains(t, field, "$type")
}

func TestBoardFieldRendersSemanticJSONAndPlainValues(t *testing.T) {
	board := domain.IssueField{
		Name: "Board", Kind: domain.FieldBoard, Cardinality: domain.CardinalityMulti,
		Value: domain.MultiValue{Values: []domain.FieldValue{
			domain.EntityValue{ID: "a-1", Name: "Alpha"}, domain.EntityValue{ID: "a-2", Name: "Beta"},
		}},
	}
	var jsonOut bytes.Buffer
	jsonRenderer, err := New(&jsonOut, true, false)
	require.NoError(t, err)
	require.NoError(t, jsonRenderer.Field(board))
	var got map[string]any
	require.NoError(t, json.Unmarshal(jsonOut.Bytes(), &got))
	assert.Equal(t, "board", got["type"])
	assert.Equal(t, "multi", got["cardinality"])
	assert.NotContains(t, got, "entityId")
	assert.Equal(t, []any{map[string]any{"entityId": "a-1", "name": "Alpha"}, map[string]any{"entityId": "a-2", "name": "Beta"}}, got["value"])

	var plainOut bytes.Buffer
	plainRenderer, err := New(&plainOut, false, true)
	require.NoError(t, err)
	require.NoError(t, plainRenderer.Field(board))
	assert.Equal(t, "Alpha, Beta\n", plainOut.String())

	empty := board
	empty.Value = domain.MultiValue{Values: make([]domain.FieldValue, 0)}
	jsonOut.Reset()
	require.NoError(t, jsonRenderer.Field(empty))
	require.NoError(t, json.Unmarshal(jsonOut.Bytes(), &got))
	assert.Equal(t, []any{}, got["value"])
	plainOut.Reset()
	require.NoError(t, plainRenderer.Field(empty))
	assert.Equal(t, "\n", plainOut.String())
}

func TestNewRejectsJSONAndPlainTogether(t *testing.T) {
	_, err := New(&bytes.Buffer{}, true, true)
	require.Error(t, err)
}

func TestRendererModesProduceIssueFieldTagAndUserContracts(t *testing.T) {
	created := time.Date(2026, 3, 4, 5, 6, 0, 0, time.UTC)
	issue := domain.Issue{
		ID: "2-1", IDReadable: "TT-1", Summary: "Summary", Description: "Description",
		Project: domain.Project{ID: "0-1", Name: "Tools", ShortName: "TT"},
		Created: created, Updated: created,
		Fields: []domain.IssueField{
			{Name: "Due", Kind: domain.FieldDate, Cardinality: domain.CardinalitySingle, Value: domain.DateValue{Value: created}},
			{Name: "Labels", Kind: domain.FieldEnum, Cardinality: domain.CardinalityMulti, Value: domain.MultiValue{Values: []domain.FieldValue{domain.EntityValue{Name: "One"}, domain.EntityValue{Name: "Two"}}}},
		},
		Tags: []domain.Tag{{ID: "t-1", Name: "backend"}},
	}
	for _, tt := range []struct {
		name  string
		json  bool
		plain bool
		check func(*testing.T, string)
	}{
		{name: "json", json: true, check: func(t *testing.T, text string) {
			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(text), &got))
			assert.Equal(t, "TT-1", got["id"])
			assert.Equal(t, "Description", got["description"])
		}},
		{name: "plain", plain: true, check: func(t *testing.T, text string) { assert.Equal(t, "TT-1\tSummary\n", text) }},
		{name: "human", check: func(t *testing.T, text string) {
			assert.Contains(t, text, "TT-1  Summary")
			assert.Contains(t, text, "Due      2026-03-04")
			assert.Contains(t, text, "Labels   One, Two")
			assert.Contains(t, text, "Tags     backend")
			assert.Contains(t, text, "Description\n")
		}},
	} {
		t.Run("issue "+tt.name, func(t *testing.T) {
			var out bytes.Buffer
			renderer, err := New(&out, tt.json, tt.plain)
			require.NoError(t, err)
			require.NoError(t, renderer.Issue(issue))
			tt.check(t, out.String())
		})
	}

	for _, tt := range []struct {
		name  string
		json  bool
		plain bool
		check func(*testing.T, string)
	}{
		{name: "json", json: true, check: func(t *testing.T, text string) { assert.Contains(t, text, `"value": "2026-03-04"`) }},
		{name: "plain", plain: true, check: func(t *testing.T, text string) { assert.Equal(t, "Due\t2026-03-04\nLabels\tOne, Two\n", text) }},
		{name: "human", check: func(t *testing.T, text string) { assert.Contains(t, text, "NAME") }},
	} {
		t.Run("fields "+tt.name, func(t *testing.T) {
			var out bytes.Buffer
			renderer, err := New(&out, tt.json, tt.plain)
			require.NoError(t, err)
			require.NoError(t, renderer.Fields(issue.Fields))
			tt.check(t, out.String())
		})
	}

	var out bytes.Buffer
	renderer, err := New(&out, false, true)
	require.NoError(t, err)
	require.NoError(t, renderer.Field(domain.IssueField{Kind: domain.FieldDateTime, Value: domain.DateValue{Value: created}}))
	assert.Equal(t, "2026-03-04T05:06:00Z\n", out.String())
	out.Reset()
	require.NoError(t, renderer.Tags(issue.Tags))
	assert.Equal(t, "backend\n", out.String())
	out.Reset()
	require.NoError(t, renderer.User(domain.User{ID: "u-1", Login: "alice"}))
	assert.Equal(t, "alice\n", out.String())
}

func TestHumanIssueOmitsEmptyDescription(t *testing.T) {
	var out bytes.Buffer
	renderer, err := New(&out, false, false)
	require.NoError(t, err)
	require.NoError(t, renderer.Issue(domain.Issue{IDReadable: "TT-1", Summary: "Summary"}))
	assert.NotContains(t, out.String(), "\n\n\n")
}
