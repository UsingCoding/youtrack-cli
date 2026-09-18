package app

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type serviceFake struct {
	issue            domain.Issue
	defs             []domain.FieldDefinition
	options          map[string][]domain.FieldOption
	searchTags       map[string][]domain.Tag
	boards           []domain.Board
	projects         []domain.Project
	getIssueErr      error
	updateErr        error
	moveErr          error
	addTagErr        error
	removeTagErr     error
	getProject       domain.Project
	getProjectErr    error
	searchProjectErr error
	searchTagsErr    error
	boardValidateErr error
	boardApplyErr    error
	getIssueCalls    int
	updateCalls      int
	moveCalls        int
	addTagCalls      int
	removeTagCalls   int
	projectSearches  int
	boardValidates   int
	boardApplies     int
	lastPatch        IssuePatch
	lastProjectID    string
	lastAddedTag     domain.Tag
	lastRemovedTag   domain.Tag
	lastAdd          []domain.Board
	lastRemove       []domain.Board
}

func (f *serviceFake) GetIssue(context.Context, domain.IssueRef) (domain.Issue, error) {
	f.getIssueCalls++
	if f.getIssueErr != nil {
		return domain.Issue{}, f.getIssueErr
	}
	return f.issue, nil
}
func (f *serviceFake) UpdateIssue(_ context.Context, _ domain.IssueRef, patch IssuePatch) error {
	f.updateCalls++
	f.lastPatch = patch
	if f.updateErr != nil {
		return f.updateErr
	}
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
func (f *serviceFake) MoveIssue(_ context.Context, _ domain.IssueRef, projectID string) (domain.Issue, error) {
	f.moveCalls++
	f.lastProjectID = projectID
	if f.moveErr != nil {
		return domain.Issue{}, f.moveErr
	}
	return f.issue, nil
}
func (f *serviceFake) AddIssueTag(_ context.Context, _ domain.IssueRef, tag domain.Tag) error {
	f.addTagCalls++
	f.lastAddedTag = tag
	return f.addTagErr
}
func (f *serviceFake) RemoveIssueTag(_ context.Context, _ domain.IssueRef, tag domain.Tag) error {
	f.removeTagCalls++
	f.lastRemovedTag = tag
	return f.removeTagErr
}
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
	if f.getProjectErr != nil {
		return domain.Project{}, f.getProjectErr
	}
	if f.getProject.ID != "" {
		return f.getProject, nil
	}
	return domain.Project{}, NotFoundf("missing")
}
func (f *serviceFake) SearchProjects(context.Context, string) ([]domain.Project, error) {
	f.projectSearches++
	return f.projects, f.searchProjectErr
}
func (f *serviceFake) SearchTags(_ context.Context, q string) ([]domain.Tag, error) {
	return f.searchTags[q], f.searchTagsErr
}
func (f *serviceFake) Me(context.Context) (domain.User, error)                      { return domain.User{}, nil }
func (f *serviceFake) SearchGroups(context.Context, string) ([]domain.Group, error) { return nil, nil }

func (f *serviceFake) ListBoards(context.Context) ([]domain.Board, error) { return f.boards, nil }
func (f *serviceFake) ValidateIssueBoardChange(_ context.Context, _ string, add, remove []domain.Board) error {
	f.boardValidates++
	f.lastAdd, f.lastRemove = add, remove
	return f.boardValidateErr
}
func (f *serviceFake) ApplyIssueBoardChange(_ context.Context, _ string, add, remove []domain.Board) error {
	f.boardApplies++
	f.lastAdd, f.lastRemove = add, remove
	return f.boardApplyErr
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

func newService(fake *serviceFake) *Service {
	return NewService(fake, nil, fake, nil, fake, fake, fake, fake, fake, fake, nil)
}

func TestServiceEnrichesFieldsFromDefinitionNameAndID(t *testing.T) {
	fake := serviceFixture()
	fake.defs = append(fake.defs,
		domain.FieldDefinition{ID: "pf-due", Name: "Due", Kind: domain.FieldDateTime, Cardinality: domain.CardinalitySingle},
		domain.FieldDefinition{ID: "pf-estimate", Name: "Estimate", Kind: domain.FieldFloat, Cardinality: domain.CardinalitySingle},
	)
	fake.issue.Fields = []domain.IssueField{
		{ID: "ignored-id", Name: "Due", Value: domain.IntegerValue{Value: 1_700_000_000_000}},
		{ID: "pf-estimate", Name: "renamed", Value: domain.IntegerValue{Value: 3}},
		{ID: "board", Name: "Board", Kind: domain.FieldBoard, Value: domain.StringValue{Value: "unchanged"}},
	}

	got, err := newService(fake).GetIssue(context.Background(), "TT-1")

	require.NoError(t, err)
	assert.Equal(t, domain.FieldDateTime, got.Fields[0].Kind)
	assert.Equal(t, time.UnixMilli(1_700_000_000_000), got.Fields[0].Value.(domain.DateValue).Value)
	assert.Equal(t, domain.FloatValue{Value: 3}, got.Fields[1].Value)
	assert.Equal(t, domain.FieldBoard, got.Fields[2].Kind)
}

func TestServiceGetFieldResolvesReferences(t *testing.T) {
	fake := serviceFixture()
	fake.issue.Fields = []domain.IssueField{
		{ID: "priority-id", Name: "Priority"},
		{ID: "component-a", Name: "Component"},
		{ID: "component-b", Name: "component"},
		{ID: "board-id", Name: "Agile", Kind: domain.FieldBoard},
	}
	service := newService(fake)

	tests := []struct {
		name string
		ref  domain.FieldRef
		id   string
		kind ErrorKind
	}{
		{name: "id", ref: "priority-id", id: "priority-id"},
		{name: "exact name", ref: "Priority", id: "priority-id"},
		{name: "case insensitive", ref: "pRiOrItY", id: "priority-id"},
		{name: "board alias", ref: " board ", id: "board-id"},
		{name: "ambiguous", ref: "COMPONENT", kind: ErrorAmbiguous},
		{name: "missing", ref: "unknown", kind: ErrorNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.GetField(context.Background(), "TT-1", tt.ref)
			if tt.kind != ErrorRuntime {
				require.Error(t, err)
				assert.Equal(t, tt.kind, KindOf(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.id, got.ID)
		})
	}
}

func TestServiceSetAndClearScalarFieldUseOnePatchAndReread(t *testing.T) {
	for _, tt := range []struct {
		name   string
		values []string
		clear  bool
		want   domain.FieldValue
	}{
		{name: "set", values: []string{"Critical"}, want: domain.EntityValue{ID: "p-critical", Name: "Critical"}},
		{name: "clear", clear: true, want: domain.EmptyValue{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fake := serviceFixture()
			service := newService(fake)
			var err error
			if tt.clear {
				_, err = service.ClearField(context.Background(), "TT-1", "Priority")
			} else {
				_, err = service.SetField(context.Background(), "TT-1", "Priority", tt.values)
			}
			require.NoError(t, err)
			assert.Equal(t, 1, fake.updateCalls)
			require.Len(t, fake.lastPatch.Fields, 1)
			assert.Equal(t, tt.want, fake.lastPatch.Fields[0].Value)
			assert.Equal(t, 2, fake.getIssueCalls, "mutation result is reread")
		})
	}
}

func TestServiceBoardChangesValidateApplyAndReread(t *testing.T) {
	newBoard := domain.Board{ID: "new", Name: "New", ProjectIDs: []string{"0-1"}}
	for _, tt := range []struct {
		name          string
		values        []string
		clear         bool
		validateErr   error
		applyErr      error
		wantValidates int
		wantApplies   int
		wantGets      int
	}{
		{name: "unchanged is no op", values: []string{"Old"}, wantGets: 1},
		{name: "set", values: []string{"New"}, wantValidates: 1, wantApplies: 1, wantGets: 2},
		{name: "clear", clear: true, wantValidates: 1, wantApplies: 1, wantGets: 2},
		{name: "validation failure", values: []string{"New"}, validateErr: errors.New("invalid"), wantValidates: 1, wantGets: 1},
		{name: "apply failure", values: []string{"New"}, applyErr: errors.New("failed"), wantValidates: 1, wantApplies: 1, wantGets: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fake := serviceFixture()
			fake.issue.Fields = []domain.IssueField{{Kind: domain.FieldBoard, Value: domain.MultiValue{Values: []domain.FieldValue{domain.EntityValue{ID: "old", Name: "Old"}}}}}
			fake.boards = []domain.Board{newBoard}
			fake.boardValidateErr = tt.validateErr
			fake.boardApplyErr = tt.applyErr
			service := newService(fake)
			var err error
			if tt.clear {
				_, err = service.ClearField(context.Background(), "TT-1", "Board")
			} else {
				_, err = service.SetField(context.Background(), "TT-1", "Board", tt.values)
			}
			if tt.validateErr != nil || tt.applyErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantValidates, fake.boardValidates)
			assert.Equal(t, tt.wantApplies, fake.boardApplies)
			assert.Equal(t, tt.wantGets, fake.getIssueCalls)
		})
	}
}

func TestServiceMoveIssueResolvesProjects(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ref      domain.ProjectRef
		get      domain.Project
		projects []domain.Project
		kind     ErrorKind
		wantID   string
	}{
		{name: "id", ref: "0-2", get: domain.Project{ID: "0-2"}, wantID: "0-2"},
		{name: "exact name", ref: "Tools", projects: []domain.Project{{ID: "0-2", Name: "Tools"}}, wantID: "0-2"},
		{name: "case insensitive", ref: "tools", projects: []domain.Project{{ID: "0-2", Name: "Tools"}}, wantID: "0-2"},
		{name: "ambiguous", ref: "tools", projects: []domain.Project{{ID: "0-2", Name: "Tools"}, {ID: "0-3", Name: "TOOLS"}}, kind: ErrorAmbiguous},
		{name: "missing", ref: "tools", kind: ErrorNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fake := serviceFixture()
			fake.getProject, fake.projects = tt.get, tt.projects
			_, err := newService(fake).MoveIssue(context.Background(), "TT-1", tt.ref)
			if tt.kind != ErrorRuntime {
				require.Error(t, err)
				assert.Equal(t, tt.kind, KindOf(err))
				assert.Equal(t, 0, fake.moveCalls)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, fake.lastProjectID)
			assert.Equal(t, 1, fake.moveCalls)
		})
	}
}

func TestServiceTagMutationsAvoidUnneededWritesAndResolveTags(t *testing.T) {
	t.Run("existing add and missing remove are no ops", func(t *testing.T) {
		fake := serviceFixture()
		service := newService(fake)
		require.NoError(t, service.AddTag(context.Background(), "TT-1", "LEGACY"))
		require.NoError(t, service.RemoveTag(context.Background(), "TT-1", "missing"))
		assert.Zero(t, fake.addTagCalls)
		assert.Zero(t, fake.removeTagCalls)
	})
	t.Run("add and remove use resolved tags", func(t *testing.T) {
		fake := serviceFixture()
		fake.searchTags["Backend"] = []domain.Tag{{ID: "tag-backend", Name: "backend"}}
		service := newService(fake)
		require.NoError(t, service.AddTag(context.Background(), "TT-1", "Backend"))
		require.NoError(t, service.RemoveTag(context.Background(), "TT-1", "legacy"))
		assert.Equal(t, domain.Tag{ID: "tag-backend", Name: "backend"}, fake.lastAddedTag)
		assert.Equal(t, domain.Tag{ID: "tag-old", Name: "legacy"}, fake.lastRemovedTag)
		assert.Equal(t, 1, fake.addTagCalls)
		assert.Equal(t, 1, fake.removeTagCalls)
	})
}
