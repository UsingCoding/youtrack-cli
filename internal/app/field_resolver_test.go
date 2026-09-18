package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type resolverFieldStore struct {
	defs            []domain.FieldDefinition
	options         map[string][]domain.FieldOption
	users           map[string][]domain.User
	defsErr         error
	optionsErr      error
	usersErr        error
	definitionCalls int
	optionCalls     int
	userCalls       int
}

func (f *resolverFieldStore) ListProjectFields(context.Context, string) ([]domain.FieldDefinition, error) {
	f.definitionCalls++
	return f.defs, f.defsErr
}
func (f *resolverFieldStore) ListFieldOptions(_ context.Context, _ string, field domain.FieldDefinition) ([]domain.FieldOption, error) {
	f.optionCalls++
	return f.options[field.ID], f.optionsErr
}
func (f *resolverFieldStore) ListFieldUsers(_ context.Context, _ string, field domain.FieldDefinition) ([]domain.User, error) {
	f.userCalls++
	return f.users[field.ID], f.usersErr
}

type resolverUserStore struct {
	me    domain.User
	err   error
	calls *int
}

func (f resolverUserStore) Me(context.Context) (domain.User, error) {
	if f.calls != nil {
		*f.calls++
	}
	return f.me, f.err
}

type resolverGroupStore struct {
	groups []domain.Group
	err    error
	calls  *int
}

func (f resolverGroupStore) SearchGroups(context.Context, string) ([]domain.Group, error) {
	if f.calls != nil {
		*f.calls++
	}
	return f.groups, f.err
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

func TestFieldResolverResolveForProject(t *testing.T) {
	resolver, issue := newResolverFixture()

	priority, err := resolver.ResolveForProject(context.Background(), issue.Project, "Priority", []string{"Critical"})
	require.NoError(t, err)
	assert.Equal(t, domain.EntityValue{ID: "p-critical", Name: "Critical"}, priority.Value)

	versions, err := resolver.ResolveForProject(context.Background(), issue.Project, "Fix versions", []string{"2026.2", "2026.3"})
	require.NoError(t, err)
	assert.Equal(t, domain.MultiValue{Values: []domain.FieldValue{
		domain.EntityValue{ID: "v-2026-2", Name: "2026.2"},
		domain.EntityValue{ID: "v-2026-3", Name: "2026.3"},
	}}, versions.Value)

	assignee, err := resolver.ResolveForProject(context.Background(), issue.Project, "Assignee", []string{"@me"})
	require.NoError(t, err)
	assert.Equal(t, domain.UserValue{ID: "u-1", Login: "john", FullName: "John Doe"}, assignee.Value)

	group, err := resolver.ResolveForProject(context.Background(), issue.Project, "Team", []string{"Platform"})
	require.NoError(t, err)
	assert.Equal(t, domain.EntityValue{ID: "g-1", Name: "Platform"}, group.Value)

	cleared, err := resolver.ResolveForProject(context.Background(), issue.Project, "Priority", []string{"@none"})
	require.NoError(t, err)
	assert.Equal(t, domain.EmptyValue{}, cleared.Value)
}

func TestFieldResolverResolveForProjectPreservesEmptyStringAndRejectsState(t *testing.T) {
	fields := &resolverFieldStore{
		defs: []domain.FieldDefinition{
			{ID: "pf-title", Name: "Title", Kind: domain.FieldString, Cardinality: domain.CardinalitySingle},
			{ID: "pf-state", Name: "State", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle},
		},
		options: map[string][]domain.FieldOption{"pf-state": {{ID: "open", Name: "Open"}}},
		users:   map[string][]domain.User{},
	}
	resolver := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{})
	project := domain.Project{ID: "0-1"}

	title, err := resolver.ResolveForProject(context.Background(), project, "Title", []string{""})
	require.NoError(t, err)
	assert.Equal(t, domain.StringValue{Value: ""}, title.Value)

	_, err = resolver.ResolveForProject(context.Background(), project, "State", []string{"Open"})
	require.Error(t, err)
	assert.Equal(t, ErrorValidation, KindOf(err))
	assert.Zero(t, fields.optionCalls)
}

func TestFieldResolverConvertsScalarValues(t *testing.T) {
	tests := []struct {
		name  string
		kind  domain.FieldKind
		input string
		want  domain.FieldValue
	}{
		{name: "string", kind: domain.FieldString, input: "value", want: domain.StringValue{Value: "value"}},
		{name: "text", kind: domain.FieldText, input: "long value", want: domain.TextValue{Value: "long value"}},
		{name: "integer", kind: domain.FieldInteger, input: "-42", want: domain.IntegerValue{Value: -42}},
		{name: "float", kind: domain.FieldFloat, input: "3.5", want: domain.FloatValue{Value: 3.5}},
		{name: "date", kind: domain.FieldDate, input: "2026-03-04", want: domain.DateValue{Value: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)}},
		{name: "date time", kind: domain.FieldDateTime, input: "2026-03-04T05:06:07Z", want: domain.DateValue{Value: time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)}},
		{name: "period", kind: domain.FieldPeriod, input: "2h15m", want: domain.PeriodValue{Minutes: 135}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := domain.FieldDefinition{ID: "field", Name: "Field", Kind: tt.kind, Cardinality: domain.CardinalitySingle}
			fields := &resolverFieldStore{defs: []domain.FieldDefinition{def}, options: map[string][]domain.FieldOption{}, users: map[string][]domain.User{}}
			resolver := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{})
			got, err := resolver.Resolve(context.Background(), domain.Issue{Project: domain.Project{ID: "0-1"}}, "Field", []string{tt.input}, false)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Value)
		})
	}
}

func TestFieldResolverRejectsInvalidScalarValues(t *testing.T) {
	for _, tt := range []struct {
		name  string
		kind  domain.FieldKind
		input string
	}{
		{name: "integer", kind: domain.FieldInteger, input: "one"},
		{name: "float", kind: domain.FieldFloat, input: "number"},
		{name: "date", kind: domain.FieldDate, input: "2026/03/04"},
		{name: "date time", kind: domain.FieldDateTime, input: "2026-03-04"},
		{name: "period empty", kind: domain.FieldPeriod, input: ""},
		{name: "period number only", kind: domain.FieldPeriod, input: "10"},
		{name: "period missing number", kind: domain.FieldPeriod, input: "h"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			def := domain.FieldDefinition{ID: "field", Name: "Field", Kind: tt.kind, Cardinality: domain.CardinalitySingle}
			fields := &resolverFieldStore{defs: []domain.FieldDefinition{def}, options: map[string][]domain.FieldOption{}, users: map[string][]domain.User{}}
			_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{}).Resolve(context.Background(), domain.Issue{Project: domain.Project{ID: "0-1"}}, "Field", []string{tt.input}, false)
			require.Error(t, err)
			assert.Equal(t, ErrorValidation, KindOf(err))
		})
	}
}

func TestFieldResolverResolvesEntityUserAndGroupReferences(t *testing.T) {
	defs := []domain.FieldDefinition{
		{ID: "option", Name: "Option", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle},
		{ID: "user", Name: "User", Kind: domain.FieldUser, Cardinality: domain.CardinalitySingle},
		{ID: "group", Name: "Group", Kind: domain.FieldGroup, Cardinality: domain.CardinalitySingle},
	}
	fields := &resolverFieldStore{
		defs:    defs,
		options: map[string][]domain.FieldOption{"option": {{ID: "o-1", Name: "Blue"}}},
		users:   map[string][]domain.User{"user": {{ID: "u-1", Login: "alice", FullName: "Alice Adams"}}},
	}
	meCalls, groupCalls := 0, 0
	resolver := NewFieldResolver(fields, resolverUserStore{me: domain.User{ID: "u-1", Login: "alice"}, calls: &meCalls}, resolverGroupStore{groups: []domain.Group{{ID: "g-1", Name: "Platform"}}, calls: &groupCalls})
	issue := domain.Issue{Project: domain.Project{ID: "0-1"}}

	tests := []struct {
		name  string
		field domain.FieldRef
		input string
		want  domain.FieldValue
	}{
		{name: "option id", field: "Option", input: "o-1", want: domain.EntityValue{ID: "o-1", Name: "Blue"}},
		{name: "option case", field: "Option", input: "blue", want: domain.EntityValue{ID: "o-1", Name: "Blue"}},
		{name: "user id", field: "User", input: "u-1", want: domain.UserValue{ID: "u-1", Login: "alice", FullName: "Alice Adams"}},
		{name: "user full name case", field: "User", input: "alice adams", want: domain.UserValue{ID: "u-1", Login: "alice", FullName: "Alice Adams"}},
		{name: "current user", field: "User", input: "@me", want: domain.UserValue{ID: "u-1", Login: "alice", FullName: "Alice Adams"}},
		{name: "group id", field: "Group", input: "g-1", want: domain.EntityValue{ID: "g-1", Name: "Platform"}},
		{name: "group case", field: "Group", input: "platform", want: domain.EntityValue{ID: "g-1", Name: "Platform"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Resolve(context.Background(), issue, tt.field, []string{tt.input}, false)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Value)
		})
	}
	assert.Equal(t, 1, meCalls)
	assert.Equal(t, 2, groupCalls)
	assert.Equal(t, 1, fields.optionCalls)
	assert.Equal(t, 1, fields.userCalls)
}

func TestFieldResolverReportsLookupAndStoreFailures(t *testing.T) {
	for _, tt := range []struct {
		name    string
		defs    []domain.FieldDefinition
		options map[string][]domain.FieldOption
		users   map[string][]domain.User
		groups  []domain.Group
		ref     domain.FieldRef
		value   string
		want    ErrorKind
	}{
		{name: "absent definition", ref: "Missing", value: "x", want: ErrorNotFound},
		{name: "ambiguous definition", defs: []domain.FieldDefinition{{ID: "a", Name: "Field", Kind: domain.FieldString, Cardinality: domain.CardinalitySingle}, {ID: "b", Name: "FIELD", Kind: domain.FieldString, Cardinality: domain.CardinalitySingle}}, ref: "field", value: "x", want: ErrorAmbiguous},
		{name: "unsupported definition", defs: []domain.FieldDefinition{{ID: "x", Name: "Field", Kind: domain.FieldUnknown, Cardinality: domain.CardinalitySingle}}, ref: "Field", value: "x", want: ErrorValidation},
		{name: "absent option", defs: []domain.FieldDefinition{{ID: "x", Name: "Field", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle}}, options: map[string][]domain.FieldOption{}, ref: "Field", value: "x", want: ErrorNotFound},
		{name: "ambiguous option", defs: []domain.FieldDefinition{{ID: "x", Name: "Field", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle}}, options: map[string][]domain.FieldOption{"x": {{ID: "a", Name: "Blue"}, {ID: "b", Name: "BLUE"}}}, ref: "Field", value: "blue", want: ErrorAmbiguous},
		{name: "absent user", defs: []domain.FieldDefinition{{ID: "x", Name: "Field", Kind: domain.FieldUser, Cardinality: domain.CardinalitySingle}}, users: map[string][]domain.User{}, ref: "Field", value: "x", want: ErrorNotFound},
		{name: "absent group", defs: []domain.FieldDefinition{{ID: "x", Name: "Field", Kind: domain.FieldGroup, Cardinality: domain.CardinalitySingle}}, groups: nil, ref: "Field", value: "x", want: ErrorNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fields := &resolverFieldStore{defs: tt.defs, options: tt.options, users: tt.users}
			_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{groups: tt.groups}).Resolve(context.Background(), domain.Issue{Project: domain.Project{ID: "0-1"}}, tt.ref, []string{tt.value}, false)
			require.Error(t, err)
			assert.Equal(t, tt.want, KindOf(err))
		})
	}
	t.Run("definition store error", func(t *testing.T) {
		fields := &resolverFieldStore{defsErr: errors.New("unavailable")}
		_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{}).Resolve(context.Background(), domain.Issue{Project: domain.Project{ID: "0-1"}}, "Field", []string{"x"}, false)
		require.ErrorIs(t, err, fields.defsErr)
	})
}

func TestFieldResolverEnforcesCardinalityAndCachesLookups(t *testing.T) {
	fields := &resolverFieldStore{
		defs: []domain.FieldDefinition{
			{ID: "single", Name: "Single", Kind: domain.FieldString, Cardinality: domain.CardinalitySingle},
			{ID: "multi", Name: "Multi", Kind: domain.FieldEnum, Cardinality: domain.CardinalityMulti},
		},
		options: map[string][]domain.FieldOption{"multi": {{ID: "one", Name: "One"}}},
		users:   map[string][]domain.User{},
	}
	resolver := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{})
	issue := domain.Issue{Project: domain.Project{ID: "0-1"}}

	_, err := resolver.Resolve(context.Background(), issue, "Single", []string{"a", "b"}, false)
	require.Error(t, err)
	assert.Equal(t, ErrorValidation, KindOf(err))
	_, err = resolver.Resolve(context.Background(), issue, "Multi", nil, false)
	require.Error(t, err)
	assert.Equal(t, ErrorValidation, KindOf(err))
	_, err = resolver.Resolve(context.Background(), issue, "Multi", []string{"One"}, false)
	require.NoError(t, err)
	_, err = resolver.Resolve(context.Background(), issue, "Multi", []string{"One"}, false)
	require.NoError(t, err)
	assert.Equal(t, 1, fields.definitionCalls)
	assert.Equal(t, 1, fields.optionCalls)
}

func TestFieldResolverProjectValidationAndTransitionFailures(t *testing.T) {
	project := domain.Project{ID: "0-1"}
	t.Run("definition lookup failure", func(t *testing.T) {
		fields := &resolverFieldStore{defsErr: errors.New("unavailable")}
		_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{}).ResolveForProject(context.Background(), project, "Field", []string{"value"})
		require.ErrorIs(t, err, fields.defsErr)
	})
	t.Run("missing definition", func(t *testing.T) {
		fields := &resolverFieldStore{options: map[string][]domain.FieldOption{}, users: map[string][]domain.User{}}
		_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{}).ResolveForProject(context.Background(), project, "Field", []string{"value"})
		require.Error(t, err)
		assert.Equal(t, ErrorNotFound, KindOf(err))
	})
	t.Run("nonempty and cardinality checks", func(t *testing.T) {
		fields := &resolverFieldStore{defs: []domain.FieldDefinition{
			{ID: "required", Name: "Required", Kind: domain.FieldString, Cardinality: domain.CardinalitySingle},
			{ID: "multi", Name: "Multi", Kind: domain.FieldString, Cardinality: domain.CardinalityMulti},
		}, options: map[string][]domain.FieldOption{}, users: map[string][]domain.User{}}
		resolver := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{})
		_, err := resolver.ResolveForProject(context.Background(), project, "Required", []string{"@none"})
		require.Error(t, err)
		assert.Equal(t, ErrorValidation, KindOf(err))
		_, err = resolver.ResolveForProject(context.Background(), project, "Multi", nil)
		require.Error(t, err)
		assert.Equal(t, ErrorValidation, KindOf(err))
		_, err = resolver.ResolveForProject(context.Background(), project, "Required", []string{"one", "two"})
		require.Error(t, err)
		assert.Equal(t, ErrorValidation, KindOf(err))
	})
	t.Run("value parse failure", func(t *testing.T) {
		fields := &resolverFieldStore{defs: []domain.FieldDefinition{{ID: "integer", Name: "Integer", Kind: domain.FieldInteger, Cardinality: domain.CardinalitySingle}}, options: map[string][]domain.FieldOption{}, users: map[string][]domain.User{}}
		_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{}).ResolveForProject(context.Background(), project, "Integer", []string{"not-a-number"})
		require.Error(t, err)
		assert.Equal(t, ErrorValidation, KindOf(err))
	})
	t.Run("state machine transition missing", func(t *testing.T) {
		fields := &resolverFieldStore{defs: []domain.FieldDefinition{{ID: "state", Name: "State", Kind: domain.FieldState, Cardinality: domain.CardinalitySingle}}, options: map[string][]domain.FieldOption{}, users: map[string][]domain.User{}}
		issue := domain.Issue{Project: project, Fields: []domain.IssueField{{ID: "state", Name: "State", StateMachine: true}}}
		_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{}).Resolve(context.Background(), issue, "State", []string{"Missing"}, false)
		require.Error(t, err)
		assert.Equal(t, ErrorNotFound, KindOf(err))
	})
}

func TestFieldResolverPropagatesLookupFailures(t *testing.T) {
	project := domain.Project{ID: "0-1"}
	for _, tt := range []struct {
		name  string
		def   domain.FieldDefinition
		store resolverFieldStore
		user  resolverUserStore
		group resolverGroupStore
		value string
	}{
		{name: "option", def: domain.FieldDefinition{ID: "option", Name: "Option", Kind: domain.FieldEnum, Cardinality: domain.CardinalitySingle}, store: resolverFieldStore{optionsErr: errors.New("options")}, value: "value"},
		{name: "user", def: domain.FieldDefinition{ID: "user", Name: "User", Kind: domain.FieldUser, Cardinality: domain.CardinalitySingle}, store: resolverFieldStore{usersErr: errors.New("users")}, value: "value"},
		{name: "current user", def: domain.FieldDefinition{ID: "user", Name: "User", Kind: domain.FieldUser, Cardinality: domain.CardinalitySingle}, store: resolverFieldStore{users: map[string][]domain.User{"user": {{ID: "u-1", Login: "alice"}}}}, user: resolverUserStore{err: errors.New("me")}, value: "@me"},
		{name: "group", def: domain.FieldDefinition{ID: "group", Name: "Group", Kind: domain.FieldGroup, Cardinality: domain.CardinalitySingle}, group: resolverGroupStore{err: errors.New("groups")}, value: "value"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.store.defs = []domain.FieldDefinition{tt.def}
			if tt.store.options == nil {
				tt.store.options = map[string][]domain.FieldOption{}
			}
			if tt.store.users == nil {
				tt.store.users = map[string][]domain.User{}
			}
			_, err := NewFieldResolver(&tt.store, tt.user, tt.group).Resolve(context.Background(), domain.Issue{Project: project}, domain.FieldRef(tt.def.Name), []string{tt.value}, false)
			require.Error(t, err)
			if tt.store.optionsErr != nil {
				assert.ErrorIs(t, err, tt.store.optionsErr)
			}
			if tt.store.usersErr != nil {
				assert.ErrorIs(t, err, tt.store.usersErr)
			}
			if tt.user.err != nil {
				assert.ErrorIs(t, err, tt.user.err)
			}
			if tt.group.err != nil {
				assert.ErrorIs(t, err, tt.group.err)
			}
		})
	}
}

func TestFieldResolverReportsAmbiguousPeopleGroupsAndOverflowPeriod(t *testing.T) {
	t.Run("ambiguous user", func(t *testing.T) {
		fields := &resolverFieldStore{defs: []domain.FieldDefinition{{ID: "user", Name: "User", Kind: domain.FieldUser, Cardinality: domain.CardinalitySingle}}, users: map[string][]domain.User{"user": {{ID: "u-1", Login: "alice"}, {ID: "u-2", Login: "ALICE"}}}, options: map[string][]domain.FieldOption{}}
		_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{}).Resolve(context.Background(), domain.Issue{Project: domain.Project{ID: "0-1"}}, "User", []string{"Alice"}, false)
		require.Error(t, err)
		assert.Equal(t, ErrorAmbiguous, KindOf(err))
	})
	t.Run("ambiguous group", func(t *testing.T) {
		fields := &resolverFieldStore{defs: []domain.FieldDefinition{{ID: "group", Name: "Group", Kind: domain.FieldGroup, Cardinality: domain.CardinalitySingle}}, options: map[string][]domain.FieldOption{}, users: map[string][]domain.User{}}
		_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{groups: []domain.Group{{ID: "g-1", Name: "Platform"}, {ID: "g-2", Name: "PLATFORM"}}}).Resolve(context.Background(), domain.Issue{Project: domain.Project{ID: "0-1"}}, "Group", []string{"platform"}, false)
		require.Error(t, err)
		assert.Equal(t, ErrorAmbiguous, KindOf(err))
	})
	t.Run("overflowing period", func(t *testing.T) {
		fields := &resolverFieldStore{defs: []domain.FieldDefinition{{ID: "period", Name: "Period", Kind: domain.FieldPeriod, Cardinality: domain.CardinalitySingle}}, options: map[string][]domain.FieldOption{}, users: map[string][]domain.User{}}
		_, err := NewFieldResolver(fields, resolverUserStore{}, resolverGroupStore{}).Resolve(context.Background(), domain.Issue{Project: domain.Project{ID: "0-1"}}, "Period", []string{"999999999999999999999999999999999999h"}, false)
		require.Error(t, err)
		assert.Equal(t, ErrorValidation, KindOf(err))
	})
}
