package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type serviceFake struct {
	issue          domain.Issue
	defs           []domain.FieldDefinition
	options        map[string][]domain.FieldOption
	searchTags     map[string][]domain.Tag
	boards         []domain.Board
	updateCalls    int
	boardValidates int
	boardApplies   int
	lastPatch      IssuePatch
	lastAdd        []domain.Board
	lastRemove     []domain.Board
}

func (f *serviceFake) GetIssue(context.Context, domain.IssueRef) (domain.Issue, error) {
	return f.issue, nil
}
func (f *serviceFake) UpdateIssue(_ context.Context, _ domain.IssueRef, patch IssuePatch) error {
	f.updateCalls++
	f.lastPatch = patch
	if patch.Summary != nil {
		f.issue.Summary = *patch.Summary
	}
	if patch.Description != nil {
		f.issue.Description = *patch.Description
	}
	if patch.Tags != nil {
		f.issue.Tags = append([]domain.Tag(nil), (*patch.Tags)...)
	}
	return nil
}
func (f *serviceFake) MoveIssue(context.Context, domain.IssueRef, string) (domain.Issue, error) {
	return f.issue, nil
}
func (f *serviceFake) AddIssueTag(context.Context, domain.IssueRef, domain.Tag) error    { return nil }
func (f *serviceFake) RemoveIssueTag(context.Context, domain.IssueRef, domain.Tag) error { return nil }
func (f *serviceFake) SearchIssues(context.Context, string, Page) ([]domain.IssueSummary, error) {
	return nil, nil
}
func (f *serviceFake) ListProjectFields(context.Context, string) ([]domain.FieldDefinition, error) {
	return f.defs, nil
}
func (f *serviceFake) ListFieldOptions(_ context.Context, _ string, field domain.FieldDefinition) ([]domain.FieldOption, error) {
	return f.options[field.ID], nil
}
func (f *serviceFake) ListFieldUsers(context.Context, string, domain.FieldDefinition) ([]domain.User, error) {
	return nil, nil
}
func (f *serviceFake) GetProject(context.Context, domain.ProjectRef) (domain.Project, error) {
	return domain.Project{}, NotFoundf("missing")
}
func (f *serviceFake) SearchProjects(context.Context, string) ([]domain.Project, error) {
	return nil, nil
}
func (f *serviceFake) SearchTags(_ context.Context, q string) ([]domain.Tag, error) {
	return f.searchTags[q], nil
}
func (f *serviceFake) Me(context.Context) (domain.User, error)                      { return domain.User{}, nil }
func (f *serviceFake) SearchGroups(context.Context, string) ([]domain.Group, error) { return nil, nil }

func (f *serviceFake) ListBoards(context.Context) ([]domain.Board, error) { return f.boards, nil }
func (f *serviceFake) ValidateIssueBoardChange(_ context.Context, _ string, add, remove []domain.Board) error {
	f.boardValidates++
	f.lastAdd, f.lastRemove = add, remove
	return nil
}
func (f *serviceFake) ApplyIssueBoardChange(_ context.Context, _ string, add, remove []domain.Board) error {
	f.boardApplies++
	f.lastAdd, f.lastRemove = add, remove
	return nil
}

func serviceFixture() *serviceFake {
	return &serviceFake{
		issue: domain.Issue{
			ID: "2-1", IDReadable: "TT-1", Summary: "Old",
			Project: domain.Project{ID: "0-1", Name: "Tools", ShortName: "TT"},
			Tags:    []domain.Tag{{ID: "tag-old", Name: "legacy"}},
		},
		defs: []domain.FieldDefinition{
			{ID: "pf-priority", Name: "Priority", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle, CanBeEmpty: true},
			{ID: "pf-state", Name: "State", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle, CanBeEmpty: false},
		},
		options: map[string][]domain.FieldOption{
			"pf-priority": {{ID: "p-critical", Name: "Critical"}},
			"pf-state":    {{ID: "s-open", Name: "Open"}},
		},
		searchTags: map[string][]domain.Tag{
			"backend": {{ID: "tag-backend", Name: "backend"}},
		},
	}
}

func TestEditIssueValidatesEverythingBeforeMutation(t *testing.T) {
	fake := serviceFixture()
	service := NewService(fake, nil, fake, nil, fake, fake, fake, fake, fake, fake, nil)
	newSummary := "New"

	_, err := service.EditIssue(context.Background(), "TT-1", EditRequest{
		Summary: &newSummary,
		Fields:  []FieldInput{{Name: "Priority", Value: "Critical"}, {Name: "State", Value: "Does not exist"}},
		AddTags: []string{"backend"},
	})

	require.Error(t, err)
	assert.Equal(t, 0, fake.updateCalls, "a late resolution error must not leave a partial mutation")
}

func TestEditIssueUsesOneCombinedMutation(t *testing.T) {
	fake := serviceFixture()
	service := NewService(fake, nil, fake, nil, fake, fake, fake, fake, fake, fake, nil)
	newSummary := "New"

	got, err := service.EditIssue(context.Background(), "TT-1", EditRequest{
		Summary: &newSummary,
		Fields:  []FieldInput{{Name: "Priority", Value: "Critical"}, {Name: "State", Value: "Open"}},
		AddTags: []string{"backend"}, RemoveTags: []string{"legacy"},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, fake.updateCalls)
	assert.Equal(t, "New", got.Summary)
	require.Len(t, fake.lastPatch.Fields, 2)
	require.NotNil(t, fake.lastPatch.Tags)
	assert.Equal(t, []domain.Tag{{ID: "tag-backend", Name: "backend"}}, *fake.lastPatch.Tags)
}

func TestEditIssueRejectsTagConflictBeforeMutation(t *testing.T) {
	fake := serviceFixture()
	service := NewService(fake, nil, fake, nil, fake, fake, fake, fake, fake, fake, nil)
	_, err := service.EditIssue(context.Background(), "TT-1", EditRequest{AddTags: []string{"Backend"}, RemoveTags: []string{"backend"}})
	require.Error(t, err)
	assert.Equal(t, 0, fake.updateCalls)
}
