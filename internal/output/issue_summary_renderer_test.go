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

func TestIssueSummariesJSONUsesStableSummaryContract(t *testing.T) {
	resolved := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	items := []domain.IssueSummary{
		{ID: "2-9", IDReadable: "TT-9", Summary: "first from server", Project: domain.Project{ID: "0-1", Name: "Tools", ShortName: "TT"}, Created: time.Date(2026, 3, 1, 1, 2, 3, 0, time.UTC), Updated: time.Date(2026, 3, 2, 1, 2, 3, 0, time.UTC)},
		{ID: "2-1", IDReadable: "TT-1", Summary: "second from server", Project: domain.Project{ID: "0-2", Name: "Platform", ShortName: "PL"}, Created: time.Date(2026, 3, 3, 1, 2, 3, 0, time.UTC), Updated: time.Date(2026, 3, 4, 1, 2, 3, 0, time.UTC), Resolved: &resolved},
	}
	var out bytes.Buffer
	renderer, err := New(&out, true, false)
	require.NoError(t, err)

	require.NoError(t, renderer.IssueSummaries(items))

	var got []map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 2)
	assert.Len(t, got[0], 7)
	assert.Contains(t, got[0], "id")
	assert.Contains(t, got[0], "entityId")
	assert.Contains(t, got[0], "summary")
	assert.Contains(t, got[0], "project")
	assert.Contains(t, got[0], "created")
	assert.Contains(t, got[0], "updated")
	assert.Contains(t, got[0], "resolved")
	assert.Equal(t, "TT-9", got[0]["id"])
	assert.Equal(t, "2-9", got[0]["entityId"])
	assert.Equal(t, "first from server", got[0]["summary"])
	assert.Equal(t, map[string]any{"entityId": "0-1", "name": "Tools", "shortName": "TT"}, got[0]["project"])
	assert.Equal(t, "2026-03-01T01:02:03Z", got[0]["created"])
	assert.Equal(t, "2026-03-02T01:02:03Z", got[0]["updated"])
	assert.Nil(t, got[0]["resolved"])
	assert.Equal(t, "TT-1", got[1]["id"])
	assert.Equal(t, "2026-03-04T05:06:07Z", got[1]["resolved"])
}

func TestIssueSummariesJSONRendersEmptyArray(t *testing.T) {
	var out bytes.Buffer
	renderer, err := New(&out, true, false)
	require.NoError(t, err)

	require.NoError(t, renderer.IssueSummaries(nil))

	assert.Equal(t, "[]\n", out.String())
}

func TestIssueSummariesPlainAndHumanOutput(t *testing.T) {
	items := []domain.IssueSummary{
		{IDReadable: "TT-9", Summary: "first", Project: domain.Project{ShortName: "TT"}, Updated: time.Date(2026, 3, 2, 1, 2, 0, 0, time.UTC)},
		{IDReadable: "TT-1", Summary: "second", Project: domain.Project{ShortName: "PL"}, Updated: time.Date(2026, 3, 3, 4, 5, 0, 0, time.UTC)},
	}
	var plainOut bytes.Buffer
	plain, err := New(&plainOut, false, true)
	require.NoError(t, err)
	require.NoError(t, plain.IssueSummaries(items))
	assert.Equal(t, "TT-9\tfirst\nTT-1\tsecond\n", plainOut.String())
	plainOut.Reset()
	require.NoError(t, plain.IssueSummaries(nil))
	assert.Empty(t, plainOut.String())

	var humanOut bytes.Buffer
	human, err := New(&humanOut, false, false)
	require.NoError(t, err)
	require.NoError(t, human.IssueSummaries(items))
	assert.Contains(t, humanOut.String(), "ID")
	assert.Contains(t, humanOut.String(), "PROJECT")
	assert.Contains(t, humanOut.String(), "UPDATED")
	assert.Contains(t, humanOut.String(), "SUMMARY")
}
