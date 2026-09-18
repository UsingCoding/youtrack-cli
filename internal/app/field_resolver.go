package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type FieldResolver struct {
	fields ProjectFieldStore
	users  UserStore
	groups GroupStore

	definitions map[string][]domain.FieldDefinition
	options     map[string][]domain.FieldOption
	fieldUsers  map[string][]domain.User
	me          *domain.User
}

func NewFieldResolver(fields ProjectFieldStore, users UserStore, groups GroupStore) *FieldResolver {
	return &FieldResolver{
		fields: fields, users: users, groups: groups,
		definitions: map[string][]domain.FieldDefinition{},
		options:     map[string][]domain.FieldOption{},
		fieldUsers:  map[string][]domain.User{},
	}
}

func (r *FieldResolver) Definitions(ctx context.Context, projectID string) ([]domain.FieldDefinition, error) {
	if v, ok := r.definitions[projectID]; ok {
		return v, nil
	}
	v, err := r.fields.ListProjectFields(ctx, projectID)
	if err != nil {
		return nil, err
	}
	r.definitions[projectID] = v
	return v, nil
}

func (r *FieldResolver) Resolve(ctx context.Context, issue domain.Issue, ref domain.FieldRef, values []string, clear bool) (FieldAssignment, error) {
	defs, err := r.Definitions(ctx, issue.Project.ID)
	if err != nil {
		return FieldAssignment{}, err
	}
	def, err := resolveDefinition(defs, string(ref))
	if err != nil {
		return FieldAssignment{}, err
	}
	if clear || (len(values) == 1 && values[0] == "@none") {
		if !def.CanBeEmpty {
			return FieldAssignment{}, Validationf("field %q cannot be empty", def.Name)
		}
		if issueField, ok := findIssueField(issue, def); ok && issueField.StateMachine {
			return FieldAssignment{}, Validationf("state-machine field %q cannot be cleared directly; use an available transition", def.Name)
		}
		return FieldAssignment{Field: def, Value: domain.EmptyValue{}}, nil
	}
	if def.Cardinality == domain.CardinalitySingle && len(values) != 1 {
		return FieldAssignment{}, Validationf("field %q is single-valued and expects exactly one value", def.Name)
	}
	if def.Cardinality == domain.CardinalityMulti && len(values) == 0 {
		return FieldAssignment{}, Validationf("field %q expects at least one value; use field clear to remove all values", def.Name)
	}

	parsed := make([]domain.FieldValue, 0, len(values))
	for _, input := range values {
		v, err := r.resolveOne(ctx, issue, def, input)
		if err != nil {
			return FieldAssignment{}, err
		}
		parsed = append(parsed, v)
	}
	if def.Cardinality == domain.CardinalityMulti {
		return FieldAssignment{Field: def, Value: domain.MultiValue{Values: parsed}}, nil
	}
	return FieldAssignment{Field: def, Value: parsed[0]}, nil
}

func (r *FieldResolver) ResolveForProject(ctx context.Context, project domain.Project, ref domain.FieldRef, values []string) (FieldAssignment, error) {
	defs, err := r.Definitions(ctx, project.ID)
	if err != nil {
		return FieldAssignment{}, err
	}
	def, err := resolveDefinition(defs, string(ref))
	if err != nil {
		return FieldAssignment{}, err
	}
	if def.Kind == domain.FieldState {
		return FieldAssignment{}, Validationf("state field %q cannot be set when creating an issue", def.Name)
	}
	if len(values) == 1 && values[0] == "@none" {
		if !def.CanBeEmpty {
			return FieldAssignment{}, Validationf("field %q cannot be empty", def.Name)
		}
		return FieldAssignment{Field: def, Value: domain.EmptyValue{}}, nil
	}
	if def.Cardinality == domain.CardinalitySingle && len(values) != 1 {
		return FieldAssignment{}, Validationf("field %q is single-valued and expects exactly one value", def.Name)
	}
	if def.Cardinality == domain.CardinalityMulti && len(values) == 0 {
		return FieldAssignment{}, Validationf("field %q expects at least one value", def.Name)
	}

	parsed := make([]domain.FieldValue, 0, len(values))
	for _, input := range values {
		v, err := r.resolveProjectValue(ctx, project.ID, def, input)
		if err != nil {
			return FieldAssignment{}, err
		}
		parsed = append(parsed, v)
	}
	if def.Cardinality == domain.CardinalityMulti {
		return FieldAssignment{Field: def, Value: domain.MultiValue{Values: parsed}}, nil
	}
	return FieldAssignment{Field: def, Value: parsed[0]}, nil
}

func (r *FieldResolver) resolveOne(ctx context.Context, issue domain.Issue, def domain.FieldDefinition, input string) (domain.FieldValue, error) {
	if def.Kind == domain.FieldState {
		if issueField, ok := findIssueField(issue, def); ok && issueField.StateMachine {
			event, err := resolveOption(issueField.Transitions, input, def.Name+" transition")
			if err != nil {
				return nil, err
			}
			return domain.StateTransitionValue{ID: event.ID, Presentation: event.Name}, nil
		}
	}
	return r.resolveProjectValue(ctx, issue.Project.ID, def, input)
}

func (r *FieldResolver) resolveProjectValue(ctx context.Context, projectID string, def domain.FieldDefinition, input string) (domain.FieldValue, error) {
	switch def.Kind {
	case domain.FieldString:
		return domain.StringValue{Value: input}, nil
	case domain.FieldText:
		return domain.TextValue{Value: input}, nil
	case domain.FieldInteger:
		v, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return nil, Validationf("field %q expects an integer: %q", def.Name, input)
		}
		return domain.IntegerValue{Value: v}, nil
	case domain.FieldFloat:
		v, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return nil, Validationf("field %q expects a number: %q", def.Name, input)
		}
		return domain.FloatValue{Value: v}, nil
	case domain.FieldDate:
		v, err := time.Parse("2006-01-02", input)
		if err != nil {
			return nil, Validationf("field %q expects YYYY-MM-DD: %q", def.Name, input)
		}
		return domain.DateValue{Value: v}, nil
	case domain.FieldDateTime:
		v, err := time.Parse(time.RFC3339, input)
		if err != nil {
			return nil, Validationf("field %q expects RFC3339 date-time: %q", def.Name, input)
		}
		return domain.DateValue{Value: v}, nil
	case domain.FieldPeriod:
		minutes, err := parsePeriod(input)
		if err != nil {
			return nil, Validationf("field %q expects a period using h/m (for example 1h30m): %q", def.Name, input)
		}
		return domain.PeriodValue{Minutes: minutes}, nil
	case domain.FieldState, domain.FieldEnum, domain.FieldVersion, domain.FieldBuild, domain.FieldOwned:
		options, err := r.optionsFor(ctx, projectID, def)
		if err != nil {
			return nil, err
		}
		opt, err := resolveOption(options, input, def.Name)
		if err != nil {
			return nil, err
		}
		return domain.EntityValue(opt), nil
	case domain.FieldUser:
		users, err := r.usersFor(ctx, projectID, def)
		if err != nil {
			return nil, err
		}
		if input == "@me" {
			me, err := r.currentUser(ctx)
			if err != nil {
				return nil, err
			}
			matched, err := resolveUser(users, me.Login, def.Name)
			if err != nil {
				return nil, Validationf("current user %q is not an allowed value for field %q", me.Login, def.Name)
			}
			return domain.UserValue{ID: matched.ID, Login: matched.Login, FullName: matched.FullName}, nil
		}
		matched, err := resolveUser(users, input, def.Name)
		if err != nil {
			return nil, err
		}
		return domain.UserValue{ID: matched.ID, Login: matched.Login, FullName: matched.FullName}, nil
	case domain.FieldGroup:
		groups, err := r.groups.SearchGroups(ctx, input)
		if err != nil {
			return nil, err
		}
		matched, err := resolveGroup(groups, input, def.Name)
		if err != nil {
			return nil, err
		}
		return domain.EntityValue(matched), nil
	default:
		return nil, Validationf("unsupported custom field type %q for field %q", def.Kind, def.Name)
	}
}

func (r *FieldResolver) optionsFor(ctx context.Context, projectID string, def domain.FieldDefinition) ([]domain.FieldOption, error) {
	key := projectID + ":" + def.ID
	if v, ok := r.options[key]; ok {
		return v, nil
	}
	v, err := r.fields.ListFieldOptions(ctx, projectID, def)
	if err != nil {
		return nil, err
	}
	r.options[key] = v
	return v, nil
}

func (r *FieldResolver) usersFor(ctx context.Context, projectID string, def domain.FieldDefinition) ([]domain.User, error) {
	key := projectID + ":" + def.ID
	if v, ok := r.fieldUsers[key]; ok {
		return v, nil
	}
	v, err := r.fields.ListFieldUsers(ctx, projectID, def)
	if err != nil {
		return nil, err
	}
	r.fieldUsers[key] = v
	return v, nil
}

func (r *FieldResolver) currentUser(ctx context.Context) (domain.User, error) {
	if r.me != nil {
		return *r.me, nil
	}
	v, err := r.users.Me(ctx)
	if err != nil {
		return domain.User{}, err
	}
	r.me = &v
	return v, nil
}

func resolveDefinition(defs []domain.FieldDefinition, ref string) (domain.FieldDefinition, error) {
	for _, d := range defs {
		if d.ID == ref || d.Name == ref {
			return d, nil
		}
	}
	var matches []domain.FieldDefinition
	for _, d := range defs {
		if strings.EqualFold(d.Name, ref) {
			matches = append(matches, d)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return domain.FieldDefinition{}, Ambiguousf("field %q is ambiguous", ref)
	}
	return domain.FieldDefinition{}, NotFoundf("field %q was not found in project", ref)
}

func resolveOption(values []domain.FieldOption, input, field string) (domain.FieldOption, error) {
	for _, v := range values {
		if v.ID == input || v.Name == input {
			return v, nil
		}
	}
	var matches []domain.FieldOption
	for _, v := range values {
		if strings.EqualFold(v.Name, input) {
			matches = append(matches, v)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return domain.FieldOption{}, Ambiguousf("value %q is ambiguous for field %q", input, field)
	}
	return domain.FieldOption{}, NotFoundf("value %q was not found for field %q", input, field)
}

func resolveUser(values []domain.User, input, field string) (domain.User, error) {
	for _, v := range values {
		if v.ID == input || v.Login == input || v.FullName == input {
			return v, nil
		}
	}
	var matches []domain.User
	for _, v := range values {
		if strings.EqualFold(v.Login, input) || strings.EqualFold(v.FullName, input) {
			matches = append(matches, v)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return domain.User{}, Ambiguousf("user %q is ambiguous for field %q", input, field)
	}
	return domain.User{}, NotFoundf("user %q was not found among allowed values for field %q", input, field)
}

func resolveGroup(values []domain.Group, input, field string) (domain.Group, error) {
	for _, v := range values {
		if v.ID == input || v.Name == input {
			return v, nil
		}
	}
	var matches []domain.Group
	for _, v := range values {
		if strings.EqualFold(v.Name, input) {
			matches = append(matches, v)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return domain.Group{}, Ambiguousf("group %q is ambiguous for field %q", input, field)
	}
	return domain.Group{}, NotFoundf("group %q was not found for field %q", input, field)
}

func findIssueField(issue domain.Issue, def domain.FieldDefinition) (domain.IssueField, bool) {
	for _, field := range issue.Fields {
		if field.ID == def.ID || field.Name == def.Name {
			return field, true
		}
	}
	return domain.IssueField{}, false
}

func parsePeriod(input string) (int64, error) {
	if input == "" {
		return 0, fmt.Errorf("empty")
	}
	var total int64
	var number strings.Builder
	seenUnit := false
	for _, ch := range input {
		if ch >= '0' && ch <= '9' {
			number.WriteRune(ch)
			continue
		}
		if ch != 'h' && ch != 'm' {
			return 0, fmt.Errorf("invalid unit")
		}
		if number.Len() == 0 {
			return 0, fmt.Errorf("missing number")
		}
		n, err := strconv.ParseInt(number.String(), 10, 64)
		if err != nil {
			return 0, err
		}
		if ch == 'h' {
			total += n * 60
		} else {
			total += n
		}
		number.Reset()
		seenUnit = true
	}
	if number.Len() != 0 || !seenUnit {
		return 0, fmt.Errorf("missing unit")
	}
	return total, nil
}
