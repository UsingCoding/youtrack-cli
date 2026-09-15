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

func savedSearchSummary() domain.IssueSummary {
	return domain.IssueSummary{ID: "2-9", IDReadable: "TT-9", Summary: "first", Project: domain.Project{ID: "0-1", Name: "Tools", ShortName: "TT"}, Created: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Updated: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)}
}

func TestSavedSearchJSONUsesStableCompositeContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		search domain.SavedSearch
		issues []domain.IssueSummary
	}{
		{name: "null owner empty issues", search: domain.SavedSearch{ID: "51-33", Name: "Mine", Query: "project: APP"}},
		{name: "owner and summary", search: domain.SavedSearch{ID: "51-33", Name: "Mine", Query: "project: APP", Owner: &domain.User{ID: "u-1", Login: "ada", FullName: "Ada Lovelace"}}, issues: []domain.IssueSummary{savedSearchSummary()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			renderer, err := New(&out, true, false)
			require.NoError(t, err)
			require.NoError(t, renderer.SavedSearch(tc.search, tc.issues))
			var got map[string]any
			require.NoError(t, json.Unmarshal(out.Bytes(), &got))
			assert.Len(t, got, 5)
			assert.Equal(t, "51-33", got["entityId"])
			assert.Equal(t, "Mine", got["name"])
			assert.Equal(t, "project: APP", got["query"])
			if tc.search.Owner == nil {
				assert.Nil(t, got["owner"])
				assert.Equal(t, []any{}, got["issues"])
				return
			}
			assert.Equal(t, map[string]any{"entityId": "u-1", "login": "ada", "name": "Ada Lovelace"}, got["owner"])
			issues := got["issues"].([]any)
			require.Len(t, issues, 1)
			assert.Nil(t, issues[0].(map[string]any)["resolved"])
		})
	}
}

func TestSavedSearchPlainAndHumanOutput(t *testing.T) {
	search := domain.SavedSearch{Name: "Mine", Query: "project: APP", Owner: &domain.User{FullName: "Ada Lovelace", Login: "ada"}}
	items := []domain.IssueSummary{savedSearchSummary(), {IDReadable: "TT-1", Summary: "second"}}
	var plainOut bytes.Buffer
	plain, err := New(&plainOut, false, true)
	require.NoError(t, err)
	require.NoError(t, plain.SavedSearch(search, items))
	assert.Equal(t, "TT-9\tfirst\nTT-1\tsecond\n", plainOut.String())
	plainOut.Reset()
	require.NoError(t, plain.SavedSearch(search, nil))
	assert.Empty(t, plainOut.String())

	var humanOut bytes.Buffer
	human, err := New(&humanOut, false, false)
	require.NoError(t, err)
	require.NoError(t, human.SavedSearch(search, items))
	got := humanOut.String()
	assert.Contains(t, got, "Name: Mine")
	assert.Contains(t, got, "Query: project: APP")
	assert.Contains(t, got, "Owner: Ada Lovelace (ada)")
	assert.Less(t, bytes.Index([]byte(got), []byte("Name: Mine")), bytes.Index([]byte(got), []byte("ID")))
	assert.Contains(t, got, "PROJECT")
	assert.Contains(t, got, "UPDATED")
	assert.Contains(t, got, "SUMMARY")
}
