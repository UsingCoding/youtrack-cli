package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func workItemForRender() domain.WorkItem {
	updated := time.Date(2026, 9, 29, 13, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	return domain.WorkItem{ID: "115-7", Date: time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), DurationMinutes: 90, Text: new("note"), Author: &domain.User{ID: "u", Login: "alice", FullName: "Alice"}, Creator: &domain.User{ID: "c", Login: "creator", FullName: "Creator"}, Type: &domain.WorkItemType{ID: "t", Name: "Development"}, Created: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC), Updated: &updated}
}

func TestWorkItemRendererStableJSONAndEmptyList(t *testing.T) {
	out := &bytes.Buffer{}
	renderer, err := New(out, true, false)
	require.NoError(t, err)
	require.NoError(t, renderer.WorkItems([]domain.WorkItem{workItemForRender(), {ID: "115-8", Date: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)}}))
	var items []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out.Bytes(), &items))
	require.Len(t, items, 2)
	assert.Equal(t, []string{"author", "created", "creator", "date", "durationMinutes", "entityId", "text", "type", "updated"}, sortedKeys(items[0]))
	assert.JSONEq(t, `"2026-09-29"`, string(items[0]["date"]))
	assert.Contains(t, string(items[0]["updated"]), "2026-09-29T11:00:00Z")
	assert.JSONEq(t, `null`, string(items[1]["text"]))
	assert.JSONEq(t, `null`, string(items[1]["author"]))
	assert.JSONEq(t, `null`, string(items[1]["creator"]))
	assert.JSONEq(t, `null`, string(items[1]["type"]))
	assert.JSONEq(t, `null`, string(items[1]["updated"]))
	out.Reset()
	require.NoError(t, renderer.WorkItems(nil))
	assert.Equal(t, "[]\n", out.String())
}

func TestWorkItemRendererPlainAndPermanentRemoval(t *testing.T) {
	out := &bytes.Buffer{}
	plain, err := New(out, false, true)
	require.NoError(t, err)
	require.NoError(t, plain.WorkItemTypes([]domain.WorkItemType{{ID: "t"}}))
	require.NoError(t, plain.WorkItems([]domain.WorkItem{workItemForRender()}))
	require.NoError(t, plain.WorkItemRemoval("115-7"))
	assert.Equal(t, "t\n115-7\n", out.String())
	out.Reset()
	human, err := New(out, false, false)
	require.NoError(t, err)
	require.NoError(t, human.WorkItemRemoval("115-7"))
	assert.Contains(t, strings.ToLower(out.String()), "permanently deleted")
	assert.NotContains(t, strings.ToLower(out.String()), "reversible")
}
