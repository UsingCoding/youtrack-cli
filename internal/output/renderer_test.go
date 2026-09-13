package output

import (
	"bytes"
	"encoding/json"
	"testing"

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
