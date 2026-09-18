package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type createProjectStoreFake struct {
	project domain.Project
	calls   *[]string
}

func (f createProjectStoreFake) GetProject(context.Context, domain.ProjectRef) (domain.Project, error) {
	*f.calls = append(*f.calls, "project")
	return f.project, nil
}
func (createProjectStoreFake) SearchProjects(context.Context, string) ([]domain.Project, error) {
	return nil, nil
}

type createFieldStoreFake struct {
	defs    []domain.FieldDefinition
	options map[string][]domain.FieldOption
	calls   *[]string
}

func (f createFieldStoreFake) ListProjectFields(context.Context, string) ([]domain.FieldDefinition, error) {
	*f.calls = append(*f.calls, "fields")
	return f.defs, nil
}
func (f createFieldStoreFake) ListFieldOptions(_ context.Context, _ string, field domain.FieldDefinition) ([]domain.FieldOption, error) {
	*f.calls = append(*f.calls, "options:"+field.Name)
	return f.options[field.ID], nil
}
func (createFieldStoreFake) ListFieldUsers(context.Context, string, domain.FieldDefinition) ([]domain.User, error) {
	return nil, nil
}

type createTagStoreFake struct {
	values map[string][]domain.Tag
	calls  *[]string
}

func (f createTagStoreFake) SearchTags(_ context.Context, ref string) ([]domain.Tag, error) {
	*f.calls = append(*f.calls, "tag:"+ref)
	return f.values[ref], nil
}

type createIssueCreatorFake struct {
	calls  []IssueCreate
	result domain.Issue
}

func (f *createIssueCreatorFake) CreateIssue(_ context.Context, create IssueCreate) (domain.Issue, error) {
	f.calls = append(f.calls, create)
	return f.result, nil
}

type createUserStoreFake struct{}

func (createUserStoreFake) Me(context.Context) (domain.User, error) { return domain.User{}, nil }

type createGroupStoreFake struct{}

func (createGroupStoreFake) SearchGroups(context.Context, string) ([]domain.Group, error) {
	return nil, nil
}

func createServiceFixture(creator *createIssueCreatorFake, calls *[]string) *Service {
	fields := createFieldStoreFake{
		defs: []domain.FieldDefinition{
			{ID: "priority", Name: "Priority", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle},
			{ID: "labels", Name: "Labels", Kind: domain.FieldEnum, Cardinality: domain.CardinalityMulti},
			{ID: "state", Name: "State", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle},
		},
		options: map[string][]domain.FieldOption{
			"priority": {{ID: "critical", Name: "Critical"}},
			"labels":   {{ID: "frontend", Name: "frontend"}, {ID: "api", Name: "api"}},
		},
		calls: calls,
	}
	tags := createTagStoreFake{values: map[string][]domain.Tag{
		"backend": {{ID: "backend-id", Name: "backend"}},
		"BACKEND": {{ID: "backend-id", Name: "backend"}},
		"missing": nil,
	}, calls: calls}
	return NewService(nil, creator, nil, nil, fields, createProjectStoreFake{project: domain.Project{ID: "0-1", Name: "App", ShortName: "APP"}, calls: calls}, tags, createUserStoreFake{}, createGroupStoreFake{}, nil, nil)
}

func TestCreateIssueResolvesInputsBeforeOneCreatorCall(t *testing.T) {
	var calls []string
	creator := &createIssueCreatorFake{result: domain.Issue{IDReadable: "APP-1", Summary: "  Preserve summary  "}}
	service := createServiceFixture(creator, &calls)
	description := "first\n\tsecond\n"

	issue, err := service.CreateIssue(context.Background(), CreateIssueRequest{
		Project: "APP", Summary: "  Preserve summary  ", Description: &description,
		Fields: []FieldInput{{Name: "Priority", Value: "Critical"}, {Name: "labels", Value: "frontend"}, {Name: " Labels ", Value: "api"}},
		Tags:   []string{"backend", "BACKEND"},
	})

	require.NoError(t, err)
	assert.Equal(t, creator.result, issue)
	require.Len(t, creator.calls, 1)
	create := creator.calls[0]
	assert.Equal(t, domain.Project{ID: "0-1", Name: "App", ShortName: "APP"}, create.Project)
	assert.Equal(t, "  Preserve summary  ", create.Summary)
	require.NotNil(t, create.Description)
	assert.Equal(t, description, *create.Description)
	assert.Equal(t, []FieldAssignment{
		{Field: domain.FieldDefinition{ID: "priority", Name: "Priority", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle}, Value: domain.EntityValue{ID: "critical", Name: "Critical"}},
		{Field: domain.FieldDefinition{ID: "labels", Name: "Labels", Kind: domain.FieldEnum, Cardinality: domain.CardinalityMulti}, Value: domain.MultiValue{Values: []domain.FieldValue{domain.EntityValue{ID: "frontend", Name: "frontend"}, domain.EntityValue{ID: "api", Name: "api"}}}},
	}, create.Fields)
	assert.Equal(t, []domain.Tag{{ID: "backend-id", Name: "backend"}}, create.Tags)
	assert.Equal(t, "project", calls[0])
	assert.Contains(t, calls, "tag:backend")
	assert.Contains(t, calls, "tag:BACKEND")
}

func TestCreateIssueRejectsInvalidInputsBeforeCreation(t *testing.T) {
	cases := []struct {
		name string
		req  CreateIssueRequest
	}{
		{"blank project", CreateIssueRequest{Project: " ", Summary: "New"}},
		{"blank summary", CreateIssueRequest{Project: "APP", Summary: " \t"}},
		{"empty field name", CreateIssueRequest{Project: "APP", Summary: "New", Fields: []FieldInput{{Name: " ", Value: "x"}}}},
		{"board", CreateIssueRequest{Project: "APP", Summary: "New", Fields: []FieldInput{{Name: "Board", Value: "x"}}}},
		{"state", CreateIssueRequest{Project: "APP", Summary: "New", Fields: []FieldInput{{Name: "State", Value: "Open"}}}},
		{"repeated scalar", CreateIssueRequest{Project: "APP", Summary: "New", Fields: []FieldInput{{Name: "Priority", Value: "Critical"}, {Name: "priority", Value: "Critical"}}}},
		{"late missing tag", CreateIssueRequest{Project: "APP", Summary: "New", Fields: []FieldInput{{Name: "Priority", Value: "Critical"}}, Tags: []string{"missing"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			creator := &createIssueCreatorFake{}
			service := createServiceFixture(creator, &calls)

			_, err := service.CreateIssue(context.Background(), tc.req)

			require.Error(t, err)
			assert.Zero(t, len(creator.calls))
		})
	}
}
