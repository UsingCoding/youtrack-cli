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

func renderLinkRelation() domain.LinkRelation {
	return domain.LinkRelation{Type: domain.LinkType{ID: "parent", Name: "Parent", Directed: true, Outward: "parent for", Inward: "subtask of"}, Direction: domain.LinkDirectionInward, Label: "subtask of"}
}

func TestLinkRendererStableJSONShapes(t *testing.T) {
	out := &bytes.Buffer{}
	renderer, err := New(out, true, false)
	require.NoError(t, err)
	require.NoError(t, renderer.LinkTypes([]domain.LinkType{{ID: "parent", Name: "Parent", Directed: true, Aggregation: true, Outward: "parent for", Inward: "subtask of"}, {ID: "relates", Name: "Relates", Outward: "relates to"}}))
	var types []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out.Bytes(), &types))
	require.Len(t, types, 2)
	assert.Equal(t, []string{"aggregation", "directed", "entityId", "inward", "name", "outward"}, sortedKeys(types[0]))
	assert.JSONEq(t, `null`, string(types[1]["inward"]))

	out.Reset()
	list := domain.IssueLinkList{IssueID: "APP-1", Relation: renderLinkRelation(), Issues: []domain.IssueSummary{}}
	require.NoError(t, renderer.IssueLinks(list))
	var got map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, []string{"issueId", "issues", "relation"}, sortedKeys(got))
	assert.JSONEq(t, `[]`, string(got["issues"]))

	out.Reset()
	change := domain.IssueLinkChange{IssueID: "APP-1", TargetIssueID: "APP-2", Relation: renderLinkRelation(), Present: true, Changed: true}
	require.NoError(t, renderer.IssueLinkChange(change))
	got = make(map[string]json.RawMessage)
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, []string{"changed", "issueId", "present", "relation", "targetIssueId"}, sortedKeys(got))
}

func TestLinkRendererPlainAndHumanChanges(t *testing.T) {
	out := &bytes.Buffer{}
	plain, err := New(out, false, true)
	require.NoError(t, err)
	require.NoError(t, plain.LinkTypes([]domain.LinkType{{ID: "parent"}, {ID: "relates"}}))
	issue := domain.IssueSummary{ID: "2-2", IDReadable: "APP-2", Summary: "Target", Created: time.Unix(0, 0)}
	require.NoError(t, plain.IssueLinks(domain.IssueLinkList{Issues: []domain.IssueSummary{issue}}))
	require.NoError(t, plain.IssueLinkChange(domain.IssueLinkChange{Changed: true}))
	assert.Equal(t, "parent\nrelates\nAPP-2\tTarget\n", out.String())

	out.Reset()
	human, err := New(out, false, false)
	require.NoError(t, err)
	require.NoError(t, human.IssueLinks(domain.IssueLinkList{Relation: renderLinkRelation()}))
	require.NoError(t, human.IssueLinkChange(domain.IssueLinkChange{IssueID: "APP-1", TargetIssueID: "APP-2", Relation: renderLinkRelation(), Present: true, Changed: false}))
	require.NoError(t, human.IssueLinkChange(domain.IssueLinkChange{IssueID: "APP-1", TargetIssueID: "APP-2", Relation: renderLinkRelation(), Present: false, Changed: true}))
	assert.Contains(t, out.String(), "Relation: subtask of (inward)")
	assert.Contains(t, out.String(), "Link already exists")
	assert.Contains(t, out.String(), "Removed link")
}
