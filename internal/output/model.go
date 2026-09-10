package output

import (
	"fmt"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type ProjectJSON struct {
	EntityID  string `json:"entityId"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type UserJSON struct {
	EntityID string `json:"entityId"`
	Login    string `json:"login"`
	Name     string `json:"name,omitempty"`
}

type TagJSON struct {
	EntityID string `json:"entityId,omitempty"`
	Name     string `json:"name"`
}

type FieldOptionJSON struct {
	EntityID string `json:"entityId"`
	Name     string `json:"name"`
}

type FieldJSON struct {
	EntityID     string            `json:"entityId,omitempty"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Cardinality  string            `json:"cardinality,omitempty"`
	Value        any               `json:"value"`
	StateMachine bool              `json:"stateMachine,omitempty"`
	Transitions  []FieldOptionJSON `json:"transitions,omitempty"`
}

type IssueJSON struct {
	ID          string      `json:"id"`
	EntityID    string      `json:"entityId"`
	Summary     string      `json:"summary"`
	Description string      `json:"description,omitempty"`
	Project     ProjectJSON `json:"project"`
	Reporter    *UserJSON   `json:"reporter,omitempty"`
	Updater     *UserJSON   `json:"updater,omitempty"`
	Created     time.Time   `json:"created"`
	Updated     time.Time   `json:"updated"`
	Resolved    *time.Time  `json:"resolved,omitempty"`
	Fields      []FieldJSON `json:"fields,omitempty"`
	Tags        []TagJSON   `json:"tags,omitempty"`
}

func issueJSON(issue domain.Issue) IssueJSON {
	out := IssueJSON{
		ID: issue.IDReadable, EntityID: issue.ID, Summary: issue.Summary, Description: issue.Description,
		Project: ProjectJSON{EntityID: issue.Project.ID, Name: issue.Project.Name, ShortName: issue.Project.ShortName},
		Created: issue.Created, Updated: issue.Updated, Resolved: issue.Resolved,
	}
	if issue.Reporter != nil {
		out.Reporter = &UserJSON{EntityID: issue.Reporter.ID, Login: issue.Reporter.Login, Name: issue.Reporter.FullName}
	}
	if issue.Updater != nil {
		out.Updater = &UserJSON{EntityID: issue.Updater.ID, Login: issue.Updater.Login, Name: issue.Updater.FullName}
	}
	for _, f := range issue.Fields {
		out.Fields = append(out.Fields, fieldJSON(f))
	}
	for _, t := range issue.Tags {
		out.Tags = append(out.Tags, TagJSON{EntityID: t.ID, Name: t.Name})
	}
	return out
}

func fieldJSON(field domain.IssueField) FieldJSON {
	out := FieldJSON{EntityID: field.ID, Name: field.Name, Type: string(field.Kind), Cardinality: string(field.Cardinality), Value: fieldValueJSON(field), StateMachine: field.StateMachine}
	for _, transition := range field.Transitions {
		out.Transitions = append(out.Transitions, FieldOptionJSON{EntityID: transition.ID, Name: transition.Name})
	}
	return out
}

func fieldValueJSON(field domain.IssueField) any {
	if v, ok := field.Value.(domain.DateValue); ok {
		if field.Kind == domain.FieldDate {
			return v.Value.Format("2006-01-02")
		}
		if field.Kind == domain.FieldDateTime {
			return v.Value.Format(time.RFC3339)
		}
	}
	return valueJSON(field.Value)
}

func valueJSON(value domain.FieldValue) any {
	switch v := value.(type) {
	case nil:
		return nil
	case domain.EmptyValue:
		return nil
	case domain.StringValue:
		return v.Value
	case domain.IntegerValue:
		return v.Value
	case domain.FloatValue:
		return v.Value
	case domain.DateValue:
		return v.Value
	case domain.PeriodValue:
		return map[string]any{"minutes": v.Minutes}
	case domain.TextValue:
		return v.Value
	case domain.EntityValue:
		return map[string]any{"entityId": v.ID, "name": v.Name}
	case domain.UserValue:
		return map[string]any{"entityId": v.ID, "login": v.Login, "name": v.FullName}
	case domain.MultiValue:
		out := make([]any, 0, len(v.Values))
		for _, item := range v.Values {
			out = append(out, valueJSON(item))
		}
		return out
	case domain.UnknownValue:
		return map[string]any{"unsupportedType": v.Type}
	default:
		return fmt.Sprint(v)
	}
}
