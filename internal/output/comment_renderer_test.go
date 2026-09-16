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

func renderComments(t *testing.T, format Format, comments []domain.Comment) string {
	t.Helper()
	out := &bytes.Buffer{}
	renderer, err := New(out, format == FormatJSON, format == FormatPlain)
	require.NoError(t, err)
	require.NoError(t, renderer.Comments(comments))
	return out.String()
}

func TestCommentRendererJSONUsesStableNullableShape(t *testing.T) {
	created := time.Date(2025, 2, 3, 4, 5, 0, 0, time.UTC)
	updated := created.Add(time.Minute)
	comments := []domain.Comment{{ID: "4-1", Author: &domain.User{ID: "u-1", Login: "ada", FullName: "Ada"}, Text: new("hello"), Created: created, Updated: &updated}, {ID: "4-2", Created: created, Deleted: true}}
	var got []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(renderComments(t, FormatJSON, comments)), &got))
	require.Len(t, got, 2)
	for _, item := range got {
		assert.Equal(t, []string{"author", "created", "deleted", "entityId", "text", "updated"}, sortedKeys(item))
	}
	assert.JSONEq(t, `null`, string(got[1]["author"]))
	assert.JSONEq(t, `null`, string(got[1]["text"]))
	assert.JSONEq(t, `null`, string(got[1]["updated"]))
	assert.JSONEq(t, `true`, string(got[1]["deleted"]))
	assert.Contains(t, string(got[0]["created"]), "2025-02-03T04:05:00Z")
	assert.JSONEq(t, `{"entityId":"u-1","login":"ada","name":"Ada"}`, string(got[0]["author"]))
	assert.Equal(t, "[]\n", renderComments(t, FormatJSON, nil))
}

func TestCommentRendererEditResponseContract(t *testing.T) {
	created := time.Date(2025, 2, 3, 4, 5, 0, 0, time.UTC)
	updated := created.Add(time.Minute)
	comment := domain.Comment{
		ID:      "4-1",
		Author:  &domain.User{Login: "ada"},
		Text:    new("revised line one\nrevised line two"),
		Created: created,
		Updated: &updated,
	}

	human := &bytes.Buffer{}
	renderer, err := New(human, false, false)
	require.NoError(t, err)
	require.NoError(t, renderer.Comment(comment))
	for _, value := range []string{"ID: 4-1", "Author: ada", "Created:", "Updated:", "Deleted: false", "revised line one\nrevised line two"} {
		assert.Contains(t, human.String(), value)
	}

	jsonOut := &bytes.Buffer{}
	renderer, err = New(jsonOut, true, false)
	require.NoError(t, err)
	require.NoError(t, renderer.Comment(domain.Comment{ID: "4-1", Created: created}))
	var object map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(jsonOut.Bytes(), &object))
	assert.Equal(t, []string{"author", "created", "deleted", "entityId", "text", "updated"}, sortedKeys(object))
	assert.JSONEq(t, `null`, string(object["author"]))
	assert.JSONEq(t, `null`, string(object["text"]))
	assert.JSONEq(t, `null`, string(object["updated"]))

	plain := &bytes.Buffer{}
	renderer, err = New(plain, false, true)
	require.NoError(t, err)
	require.NoError(t, renderer.Comment(comment))
	assert.Equal(t, "4-1\n", plain.String())
}

func TestCommentRendererPlainAndRemovalBytes(t *testing.T) {
	comments := []domain.Comment{{ID: "4-2"}, {ID: "4-1", Deleted: true}}
	assert.Equal(t, "4-2\n4-1\n", renderComments(t, FormatPlain, comments))
	out := &bytes.Buffer{}
	renderer, err := New(out, false, true)
	require.NoError(t, err)
	require.NoError(t, renderer.Comment(domain.Comment{ID: "4-3"}))
	assert.Equal(t, "4-3\n", out.String())
	out.Reset()
	require.NoError(t, renderer.CommentRemoval("4-3"))
	assert.Empty(t, out.String())
}

func TestCommentRendererHumanPreservesMetadataAndText(t *testing.T) {
	out := &bytes.Buffer{}
	renderer, err := New(out, false, false)
	require.NoError(t, err)
	created := time.Date(2025, 2, 3, 4, 5, 0, 0, time.UTC)
	require.NoError(t, renderer.Comments([]domain.Comment{{ID: "4-1", Created: created, Deleted: true, Text: new("line one\nline two")}, {ID: "4-2", Author: &domain.User{Login: "ada"}, Created: created, Text: new("end\n")}}))
	got := out.String()
	for _, label := range []string{"ID: 4-1", "Author: -", "Created:", "Updated: -", "Deleted: true", "Text:", "line one\nline two", "Author: ada"} {
		assert.Contains(t, got, label)
	}
	out.Reset()
	require.NoError(t, renderer.CommentRemoval("4-1"))
	removal := out.String()
	assert.Contains(t, removal, "removed")
	assert.Contains(t, removal, "reversible")
	assert.NotContains(t, strings.ToLower(removal), "permanently deleted")
}

func sortedKeys(item map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(item))
	for key := range item {
		keys = append(keys, key)
	}
	for i := range keys {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
