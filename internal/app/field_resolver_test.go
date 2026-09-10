package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type resolverFieldStore struct {
	defs    []domain.FieldDefinition
	options map[string][]domain.FieldOption
	users   map[string][]domain.User
}

func (f *resolverFieldStore) ListProjectFields(context.Context, string) ([]domain.FieldDefinition, error) {
	return f.defs, nil
}
func (f *resolverFieldStore) ListFieldOptions(_ context.Context, _ string, field domain.FieldDefinition) ([]domain.FieldOption, error) {
	return f.options[field.ID], nil
}
func (f *resolverFieldStore) ListFieldUsers(_ context.Context, _ string, field domain.FieldDefinition) ([]domain.User, error) {
	return f.users[field.ID], nil
}

type resolverUserStore struct{ me domain.User }

func (f resolverUserStore) Me(context.Context) (domain.User, error) { return f.me, nil }

type resolverGroupStore struct{ groups []domain.Group }

func (f resolverGroupStore) SearchGroups(context.Context, string) ([]domain.Group, error) {
	return f.groups, nil
}

func newResolverFixture() (*FieldResolver, domain.Issue) {
	fields := &resolverFieldStore{
		defs: []domain.FieldDefinition{
			{ID: "pf-priority", Name: "Priority", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle, CanBeEmpty: true},
			{ID: "pf-versions", Name: "Fix versions", Kind: domain.FieldVersion, Cardinality: domain.CardinalityMulti, CanBeEmpty: true},
			{ID: "pf-assignee", Name: "Assignee", Kind: domain.FieldUser, Cardinality: domain.CardinalitySingle, CanBeEmpty: true, BundleID: "users"},
			{ID: "pf-state", Name: "State", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle, CanBeEmpty: false},
			{ID: "pf-estimate", Name: "Estimation", Kind: domain.FieldPeriod, Cardinality: domain.CardinalitySingle, CanBeEmpty: true},
			{ID: "pf-group", Name: "Team", Kind: domain.FieldGroup, Cardinality: domain.CardinalitySingle, CanBeEmpty: true},
		},
		options: map[string][]domain.FieldOption{
			"pf-priority": {{ID: "p-critical", Name: "Critical"}, {ID: "p-major", Name: "Major"}},
			"pf-versions": {{ID: "v-2026-2", Name: "2026.2"}, {ID: "v-2026-3", Name: "2026.3"}},
			"pf-state":    {{ID: "s-open", Name: "Open"}},
		},
		users: map[string][]domain.User{
			"pf-assignee": {{ID: "u-1", Login: "john", FullName: "John Doe"}},
		},
	}
	resolver := NewFieldResolver(fields, resolverUserStore{me: domain.User{ID: "u-1", Login: "john", FullName: "John Doe"}}, resolverGroupStore{groups: []domain.Group{{ID: "g-1", Name: "Platform"}}})
	issue := domain.Issue{IDReadable: "TT-1", Project: domain.Project{ID: "0-1", ShortName: "TT"}}
	return resolver, issue
}

func TestFieldResolverEnumAndMultiValue(t *testing.T) {
	resolver, issue := newResolverFixture()

	priority, err := resolver.Resolve(context.Background(), issue, "priority", []string{"critical"}, false)
	require.NoError(t, err)
	assert.Equal(t, domain.FieldEnum, priority.Field.Kind)
	assert.Equal(t, domain.EntityValue{ID: "p-critical", Name: "Critical"}, priority.Value)

	versions, err := resolver.Resolve(context.Background(), issue, "Fix versions", []string{"2026.2", "2026.3"}, false)
	require.NoError(t, err)
	assert.Equal(t, domain.MultiValue{Values: []domain.FieldValue{
		domain.EntityValue{ID: "v-2026-2", Name: "2026.2"},
		domain.EntityValue{ID: "v-2026-3", Name: "2026.3"},
	}}, versions.Value)
}

func TestFieldResolverMeAndClear(t *testing.T) {
	resolver, issue := newResolverFixture()

	assignee, err := resolver.Resolve(context.Background(), issue, "Assignee", []string{"@me"}, false)
	require.NoError(t, err)
	assert.Equal(t, domain.UserValue{ID: "u-1", Login: "john", FullName: "John Doe"}, assignee.Value)

	cleared, err := resolver.Resolve(context.Background(), issue, "Assignee", nil, true)
	require.NoError(t, err)
	assert.Equal(t, domain.EmptyValue{}, cleared.Value)

	_, err = resolver.Resolve(context.Background(), issue, "State", nil, true)
	require.Error(t, err)
	assert.Equal(t, ErrorValidation, KindOf(err))
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestFieldResolverPeriodAndGroup(t *testing.T) {
	resolver, issue := newResolverFixture()

	estimate, err := resolver.Resolve(context.Background(), issue, "Estimation", []string{"1h30m"}, false)
	require.NoError(t, err)
	assert.Equal(t, domain.PeriodValue{Minutes: 90}, estimate.Value)

	group, err := resolver.Resolve(context.Background(), issue, "Team", []string{"platform"}, false)
	require.NoError(t, err)
	assert.Equal(t, domain.EntityValue{ID: "g-1", Name: "Platform"}, group.Value)
}

func TestFieldResolverRejectsMultipleValuesForScalar(t *testing.T) {
	resolver, issue := newResolverFixture()
	_, err := resolver.Resolve(context.Background(), issue, "Priority", []string{"Major", "Critical"}, false)
	require.Error(t, err)
	assert.Equal(t, ErrorValidation, KindOf(err))
}

func TestFieldResolverRejectsUnsupportedPeriodUnits(t *testing.T) {
	resolver, issue := newResolverFixture()
	_, err := resolver.Resolve(context.Background(), issue, "Estimation", []string{"1d"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "using h/m")
}

func TestFieldResolverUsesStateMachineTransitionsFromIssue(t *testing.T) {
	resolver, issue := newResolverFixture()
	issue.Fields = []domain.IssueField{{
		ID: "pf-state", Name: "State", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle,
		StateMachine: true,
		Transitions:  []domain.FieldOption{{ID: "in-progress", Name: "In Progress"}, {ID: "fixed", Name: "Fixed"}},
	}}

	assignment, err := resolver.Resolve(context.Background(), issue, "State", []string{"In Progress"}, false)
	require.NoError(t, err)
	assert.Equal(t, domain.StateTransitionValue{ID: "in-progress", Presentation: "In Progress"}, assignment.Value)
}

func TestFieldResolverRejectsClearingStateMachineField(t *testing.T) {
	fields := &resolverFieldStore{
		defs:    []domain.FieldDefinition{{ID: "pf-state", Name: "State", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle, CanBeEmpty: true}},
		options: map[string][]domain.FieldOption{},
		users:   map[string][]domain.User{},
	}
	resolver := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{})
	issue := domain.Issue{Project: domain.Project{ID: "0-1"}, Fields: []domain.IssueField{{ID: "pf-state", Name: "State", Kind: domain.FieldState, StateMachine: true}}}

	_, err := resolver.Resolve(context.Background(), issue, "State", nil, true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be cleared directly")
}
