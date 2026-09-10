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

func TestNewRejectsJSONAndPlainTogether(t *testing.T) {
	_, err := New(&bytes.Buffer{}, true, true)
	require.Error(t, err)
}
