package output

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func TestFieldJSONUsesStableSemanticValues(t *testing.T) {
	date := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	tests := []struct {
		name  string
		field domain.IssueField
		want  any
	}{
		{name: "empty", field: domain.IssueField{Value: domain.EmptyValue{}}, want: nil},
		{name: "string", field: domain.IssueField{Value: domain.StringValue{Value: "value"}}, want: "value"},
		{name: "integer", field: domain.IssueField{Value: domain.IntegerValue{Value: 3}}, want: int64(3)},
		{name: "float", field: domain.IssueField{Value: domain.FloatValue{Value: 3.5}}, want: 3.5},
		{name: "date", field: domain.IssueField{Kind: domain.FieldDate, Value: domain.DateValue{Value: date}}, want: "2026-03-04"},
		{name: "date time", field: domain.IssueField{Kind: domain.FieldDateTime, Value: domain.DateValue{Value: date}}, want: "2026-03-04T05:06:07Z"},
		{name: "period", field: domain.IssueField{Value: domain.PeriodValue{Minutes: 90}}, want: map[string]any{"minutes": int64(90)}},
		{name: "text", field: domain.IssueField{Value: domain.TextValue{Value: "long"}}, want: "long"},
		{name: "entity", field: domain.IssueField{Value: domain.EntityValue{ID: "e", Name: "Entity"}}, want: map[string]any{"entityId": "e", "name": "Entity"}},
		{name: "user", field: domain.IssueField{Value: domain.UserValue{ID: "u", Login: "alice", FullName: "Alice"}}, want: map[string]any{"entityId": "u", "login": "alice", "name": "Alice"}},
		{name: "multi", field: domain.IssueField{Value: domain.MultiValue{Values: []domain.FieldValue{domain.StringValue{Value: "one"}, domain.IntegerValue{Value: 2}}}}, want: []any{"one", int64(2)}},
		{name: "unknown", field: domain.IssueField{Value: domain.UnknownValue{Type: "Future"}}, want: map[string]any{"unsupportedType": "Future"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.field.ID, tt.field.Name, tt.field.Cardinality = "f-1", "Field", domain.CardinalitySingle
			got := fieldJSON(tt.field)
			assert.Equal(t, "f-1", got.EntityID)
			assert.Equal(t, string(tt.field.Kind), got.Type)
			assert.Equal(t, string(domain.CardinalitySingle), got.Cardinality)
			assert.Equal(t, tt.want, got.Value)
			data, err := json.Marshal(got)
			assert.NoError(t, err)
			assert.NotContains(t, string(data), "$type")
		})
	}
}

func TestFieldJSONIncludesStateTransitions(t *testing.T) {
	got := fieldJSON(domain.IssueField{ID: "state", Name: "State", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle, StateMachine: true, Transitions: []domain.FieldOption{{ID: "go", Name: "Go"}}})
	assert.True(t, got.StateMachine)
	assert.Equal(t, []FieldOptionJSON{{EntityID: "go", Name: "Go"}}, got.Transitions)
}
