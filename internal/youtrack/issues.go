package youtrack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack/dto"
)

func (c *Client) GetIssue(ctx context.Context, ref domain.IssueRef) (domain.Issue, error) {
	var data dto.Issue
	q := url.Values{"fields": []string{issueFields}}
	if err := c.doJSON(ctx, http.MethodGet, "/api/issues/"+string(ref), q, nil, &data); err != nil {
		return domain.Issue{}, err
	}
	issue, err := mapIssue(data)
	if err != nil {
		return domain.Issue{}, err
	}
	boards, err := c.listIssueBoards(ctx, ref)
	if err != nil {
		return domain.Issue{}, err
	}
	issue.Fields = append([]domain.IssueField{boardIssueField(boards)}, issue.Fields...)
	return issue, nil
}

func (c *Client) UpdateIssue(ctx context.Context, ref domain.IssueRef, patch app.IssuePatch) error {
	payload := map[string]any{}
	if patch.Summary != nil {
		payload["summary"] = *patch.Summary
	}
	if patch.Description != nil {
		payload["description"] = *patch.Description
	}
	if len(patch.Fields) > 0 {
		fields := make([]any, 0, len(patch.Fields))
		for _, a := range patch.Fields {
			item, err := serializeAssignment(a)
			if err != nil {
				return err
			}
			fields = append(fields, item)
		}
		payload["customFields"] = fields
	}
	if patch.Tags != nil {
		tags := make([]map[string]string, 0, len(*patch.Tags))
		for _, tag := range *patch.Tags {
			tags = append(tags, map[string]string{"id": tag.ID})
		}
		payload["tags"] = tags
	}
	return c.doJSON(ctx, http.MethodPost, "/api/issues/"+string(ref), nil, payload, nil)
}

func (c *Client) CreateIssue(ctx context.Context, create app.IssueCreate) (domain.Issue, error) {
	payload := map[string]any{
		"project": map[string]string{"id": create.Project.ID},
		"summary": create.Summary,
	}
	if create.Description != nil {
		payload["description"] = *create.Description
	}
	if len(create.Fields) > 0 {
		fields := make([]any, 0, len(create.Fields))
		for _, assignment := range create.Fields {
			field, err := serializeCreateAssignment(assignment)
			if err != nil {
				return domain.Issue{}, err
			}
			fields = append(fields, field)
		}
		payload["customFields"] = fields
	}
	if len(create.Tags) > 0 {
		tags := make([]map[string]string, 0, len(create.Tags))
		for _, tag := range create.Tags {
			tags = append(tags, map[string]string{"id": tag.ID})
		}
		payload["tags"] = tags
	}

	var data dto.Issue
	q := url.Values{"fields": []string{issueFields}}
	if err := c.doJSON(ctx, http.MethodPost, "/api/issues", q, payload, &data); err != nil {
		return domain.Issue{}, err
	}
	return mapIssue(data)
}

func (c *Client) MoveIssue(ctx context.Context, ref domain.IssueRef, projectID string) (domain.Issue, error) {
	path := "/api/issues/" + string(ref) + "/project"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, map[string]string{"id": projectID}, nil); err != nil {
		return domain.Issue{}, err
	}
	return c.GetIssue(ctx, ref)
}

func (c *Client) AddIssueTag(ctx context.Context, ref domain.IssueRef, tag domain.Tag) error {
	path := "/api/issues/" + string(ref) + "/tags"
	return c.doJSON(ctx, http.MethodPost, path, nil, map[string]string{"id": tag.ID}, nil)
}

func (c *Client) RemoveIssueTag(ctx context.Context, ref domain.IssueRef, tag domain.Tag) error {
	path := "/api/issues/" + string(ref) + "/tags/" + tag.ID
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func mapIssue(src dto.Issue) (domain.Issue, error) {
	out := domain.Issue{
		ID: src.ID, IDReadable: src.IDReadable, Summary: src.Summary, Description: src.Description,
		Project: domain.Project{ID: src.Project.ID, Name: src.Project.Name, ShortName: src.Project.ShortName},
		Created: time.UnixMilli(src.Created), Updated: time.UnixMilli(src.Updated),
	}
	if src.Resolved != nil {
		t := time.UnixMilli(*src.Resolved)
		out.Resolved = &t
	}
	if src.Reporter != nil {
		out.Reporter = &domain.User{ID: src.Reporter.ID, Login: src.Reporter.Login, FullName: src.Reporter.FullName}
	}
	if src.Updater != nil {
		out.Updater = &domain.User{ID: src.Updater.ID, Login: src.Updater.Login, FullName: src.Updater.FullName}
	}
	for _, t := range src.Tags {
		out.Tags = append(out.Tags, domain.Tag{ID: t.ID, Name: t.Name})
	}
	for _, f := range src.CustomFields {
		mapped, err := mapIssueField(f)
		if err != nil {
			return domain.Issue{}, fmt.Errorf("map custom field %q: %w", f.Name, err)
		}
		out.Fields = append(out.Fields, mapped)
	}
	return out, nil
}

func mapIssueField(src dto.IssueCustomField) (domain.IssueField, error) {
	out := domain.IssueField{ID: src.ID, Name: src.Name, Kind: kindFromIssueType(src.Type), Cardinality: cardinalityFromIssueType(src.Type), StateMachine: src.Type == "StateMachineIssueCustomField"}
	for _, event := range src.PossibleEvents {
		out.Transitions = append(out.Transitions, domain.FieldOption{ID: event.ID, Name: event.Presentation})
	}
	if string(src.Value) == "null" || len(src.Value) == 0 {
		out.Value = domain.EmptyValue{}
		return out, nil
	}
	var err error
	switch {
	case strings.HasPrefix(src.Type, "Multi"):
		out.Value, err = parseMultiEntityValue(src.Type, src.Value)
	case src.Type == "SingleUserIssueCustomField":
		out.Value, err = parseUserValue(src.Value)
	case strings.HasPrefix(src.Type, "Single") || src.Type == "StateIssueCustomField" || src.Type == "StateMachineIssueCustomField":
		out.Value, err = parseEntityValue(src.Value)
	case src.Type == "DateIssueCustomField":
		var ms int64
		err = json.Unmarshal(src.Value, &ms)
		out.Value = domain.DateValue{Value: time.UnixMilli(ms)}
	case src.Type == "PeriodIssueCustomField":
		var v struct {
			Minutes int64 `json:"minutes"`
		}
		err = json.Unmarshal(src.Value, &v)
		out.Value = domain.PeriodValue{Minutes: v.Minutes}
	case src.Type == "TextIssueCustomField":
		var v struct {
			Text string `json:"text"`
		}
		err = json.Unmarshal(src.Value, &v)
		out.Value = domain.TextValue{Value: v.Text}
	case src.Type == "SimpleIssueCustomField":
		out.Value, err = parsePrimitive(src.Value)
	default:
		out.Value = domain.UnknownValue{Type: src.Type}
	}
	return out, err
}

func parsePrimitive(raw json.RawMessage) (domain.FieldValue, error) {
	var s string
	if len(raw) > 0 && raw[0] == '"' {
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		return domain.StringValue{Value: s}, nil
	}
	text := string(raw)
	if strings.ContainsAny(text, ".eE") {
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, err
		}
		return domain.FloatValue{Value: f}, nil
	}
	var i int64
	if err := json.Unmarshal(raw, &i); err != nil {
		return nil, err
	}
	return domain.IntegerValue{Value: i}, nil
}

func parseEntityValue(raw json.RawMessage) (domain.FieldValue, error) {
	var v struct{ ID, Name string }
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return domain.EntityValue{ID: v.ID, Name: v.Name}, nil
}

func parseUserValue(raw json.RawMessage) (domain.FieldValue, error) {
	var v struct{ ID, Login, FullName string }
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return domain.UserValue{ID: v.ID, Login: v.Login, FullName: v.FullName}, nil
}

func parseMultiEntityValue(fieldType string, raw json.RawMessage) (domain.FieldValue, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	values := make([]domain.FieldValue, 0, len(items))
	for _, item := range items {
		var v domain.FieldValue
		var err error
		if fieldType == "MultiUserIssueCustomField" {
			v, err = parseUserValue(item)
		} else {
			v, err = parseEntityValue(item)
		}
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return domain.MultiValue{Values: values}, nil
}

func serializeAssignment(a app.FieldAssignment) (map[string]any, error) {
	if transition, ok := a.Value.(domain.StateTransitionValue); ok {
		return map[string]any{
			"id":    a.Field.ID,
			"$type": "StateMachineIssueCustomField",
			"event": map[string]any{"id": transition.ID, "presentation": transition.Presentation, "$type": "Event"},
		}, nil
	}
	typeName, err := issueTypeFor(a.Field)
	if err != nil {
		return nil, err
	}
	value, err := serializeValue(a.Field, a.Value)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": a.Field.ID, "$type": typeName, "value": value}, nil
}

func serializeCreateAssignment(a app.FieldAssignment) (map[string]any, error) {
	if _, ok := a.Value.(domain.StateTransitionValue); ok {
		return nil, app.Validationf("state transitions cannot be used when creating an issue")
	}
	typeName, err := issueTypeFor(a.Field)
	if err != nil {
		return nil, err
	}
	value, err := serializeValue(a.Field, a.Value)
	if err != nil {
		return nil, err
	}
	return map[string]any{"name": a.Field.Name, "$type": typeName, "value": value}, nil
}

func issueTypeFor(def domain.FieldDefinition) (string, error) {
	multi := def.Cardinality == domain.CardinalityMulti
	switch def.Kind {
	case domain.FieldEnum:
		if multi {
			return "MultiEnumIssueCustomField", nil
		}
		return "SingleEnumIssueCustomField", nil
	case domain.FieldState:
		return "StateIssueCustomField", nil
	case domain.FieldUser:
		if multi {
			return "MultiUserIssueCustomField", nil
		}
		return "SingleUserIssueCustomField", nil
	case domain.FieldVersion:
		if multi {
			return "MultiVersionIssueCustomField", nil
		}
		return "SingleVersionIssueCustomField", nil
	case domain.FieldBuild:
		if multi {
			return "MultiBuildIssueCustomField", nil
		}
		return "SingleBuildIssueCustomField", nil
	case domain.FieldOwned:
		if multi {
			return "MultiOwnedIssueCustomField", nil
		}
		return "SingleOwnedIssueCustomField", nil
	case domain.FieldGroup:
		if multi {
			return "MultiGroupIssueCustomField", nil
		}
		return "SingleGroupIssueCustomField", nil
	case domain.FieldDate:
		return "DateIssueCustomField", nil
	case domain.FieldPeriod:
		return "PeriodIssueCustomField", nil
	case domain.FieldText:
		return "TextIssueCustomField", nil
	case domain.FieldString, domain.FieldInteger, domain.FieldFloat, domain.FieldDateTime:
		return "SimpleIssueCustomField", nil
	default:
		return "", app.Validationf("unsupported custom field type %q", def.Kind)
	}
}

func serializeValue(def domain.FieldDefinition, value domain.FieldValue) (any, error) {
	if _, ok := value.(domain.EmptyValue); ok {
		if def.Cardinality == domain.CardinalityMulti {
			return []any{}, nil
		}
		return nil, nil
	}
	switch v := value.(type) {
	case domain.StringValue:
		return v.Value, nil
	case domain.IntegerValue:
		return v.Value, nil
	case domain.FloatValue:
		return v.Value, nil
	case domain.TextValue:
		return map[string]any{"text": v.Value}, nil
	case domain.PeriodValue:
		return map[string]any{"minutes": v.Minutes}, nil
	case domain.DateValue:
		return v.Value.UnixMilli(), nil
	case domain.EntityValue:
		return map[string]any{"id": v.ID}, nil
	case domain.UserValue:
		return map[string]any{"id": v.ID}, nil
	case domain.MultiValue:
		out := make([]any, 0, len(v.Values))
		for _, item := range v.Values {
			x, err := serializeValue(domain.FieldDefinition{Kind: def.Kind, Cardinality: domain.CardinalitySingle}, item)
			if err != nil {
				return nil, err
			}
			out = append(out, x)
		}
		return out, nil
	default:
		return nil, app.Validationf("unsupported value for field %q", def.Name)
	}
}

func kindFromIssueType(t string) domain.FieldKind {
	switch t {
	case "SingleEnumIssueCustomField", "MultiEnumIssueCustomField":
		return domain.FieldEnum
	case "StateIssueCustomField", "StateMachineIssueCustomField":
		return domain.FieldState
	case "SingleUserIssueCustomField", "MultiUserIssueCustomField":
		return domain.FieldUser
	case "SingleVersionIssueCustomField", "MultiVersionIssueCustomField":
		return domain.FieldVersion
	case "SingleBuildIssueCustomField", "MultiBuildIssueCustomField":
		return domain.FieldBuild
	case "SingleOwnedIssueCustomField", "MultiOwnedIssueCustomField":
		return domain.FieldOwned
	case "SingleGroupIssueCustomField", "MultiGroupIssueCustomField":
		return domain.FieldGroup
	case "DateIssueCustomField":
		return domain.FieldDate
	case "PeriodIssueCustomField":
		return domain.FieldPeriod
	case "TextIssueCustomField":
		return domain.FieldText
	default:
		return domain.FieldUnknown
	}
}

func cardinalityFromIssueType(t string) domain.Cardinality {
	if strings.HasPrefix(t, "Multi") {
		return domain.CardinalityMulti
	}
	return domain.CardinalitySingle
}
