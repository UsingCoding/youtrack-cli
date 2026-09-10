package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type Format string

const (
	FormatHuman Format = "human"
	FormatJSON  Format = "json"
	FormatPlain Format = "plain"
)

type Renderer struct {
	Out    io.Writer
	Format Format
}

func New(out io.Writer, jsonMode, plainMode bool) (*Renderer, error) {
	if jsonMode && plainMode {
		return nil, app.Validationf("--json and --plain are mutually exclusive")
	}
	format := FormatHuman
	if jsonMode {
		format = FormatJSON
	}
	if plainMode {
		format = FormatPlain
	}
	return &Renderer{Out: out, Format: format}, nil
}

func (r *Renderer) Issue(issue domain.Issue) error {
	switch r.Format {
	case FormatJSON:
		return r.json(issueJSON(issue))
	case FormatPlain:
		_, err := fmt.Fprintf(r.Out, "%s\t%s\n", issue.IDReadable, issue.Summary)
		return err
	default:
		return r.humanIssue(issue)
	}
}

func (r *Renderer) Fields(fields []domain.IssueField) error {
	if r.Format == FormatJSON {
		out := make([]FieldJSON, 0, len(fields))
		for _, f := range fields {
			out = append(out, fieldJSON(f))
		}
		return r.json(out)
	}
	if r.Format == FormatPlain {
		for _, f := range fields {
			if _, err := fmt.Fprintf(r.Out, "%s\t%s\n", f.Name, fieldPlainValue(f)); err != nil {
				return err
			}
		}
		return nil
	}
	w := tabwriter.NewWriter(r.Out, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "NAME\tTYPE\tVALUE"); err != nil {
		return err
	}
	for _, f := range fields {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", f.Name, friendlyType(f), fieldPlainValue(f)); err != nil {
			return err
		}
	}
	return w.Flush()
}

func (r *Renderer) Field(field domain.IssueField) error {
	if r.Format == FormatJSON {
		return r.json(fieldJSON(field))
	}
	if r.Format == FormatPlain {
		_, err := fmt.Fprintln(r.Out, fieldPlainValue(field))
		return err
	}
	w := tabwriter.NewWriter(r.Out, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintf(w, "Name\t%s\nType\t%s\nValue\t%s\n", field.Name, friendlyType(field), fieldPlainValue(field)); err != nil {
		return err
	}
	return w.Flush()
}

func (r *Renderer) Tags(tags []domain.Tag) error {
	if r.Format == FormatJSON {
		out := make([]TagJSON, 0, len(tags))
		for _, t := range tags {
			out = append(out, TagJSON{EntityID: t.ID, Name: t.Name})
		}
		return r.json(out)
	}
	for _, t := range tags {
		if _, err := fmt.Fprintln(r.Out, t.Name); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) User(user domain.User) error {
	if r.Format == FormatJSON {
		return r.json(UserJSON{EntityID: user.ID, Login: user.Login, Name: user.FullName})
	}
	if r.Format == FormatPlain {
		_, err := fmt.Fprintln(r.Out, user.Login)
		return err
	}
	_, err := fmt.Fprintf(r.Out, "%s (%s)\n", user.FullName, user.Login)
	return err
}

func (r *Renderer) JSON(v any) error { return r.json(v) }

func (r *Renderer) json(v any) error {
	enc := json.NewEncoder(r.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (r *Renderer) humanIssue(issue domain.Issue) error {
	if _, err := fmt.Fprintf(r.Out, "%s  %s\n\n", issue.IDReadable, issue.Summary); err != nil {
		return err
	}
	w := tabwriter.NewWriter(r.Out, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintf(w, "Project\t%s\n", issue.Project.ShortName); err != nil {
		return err
	}
	for _, f := range issue.Fields {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", f.Name, fieldPlainValue(f)); err != nil {
			return err
		}
	}
	if len(issue.Tags) > 0 {
		names := make([]string, 0, len(issue.Tags))
		for _, t := range issue.Tags {
			names = append(names, t.Name)
		}
		if _, err := fmt.Fprintf(w, "Tags\t%s\n", strings.Join(names, ", ")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "\nCreated\t%s\nUpdated\t%s\n", issue.Created.Format("2006-01-02 15:04"), issue.Updated.Format("2006-01-02 15:04")); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if issue.Description != "" {
		if _, err := fmt.Fprintf(r.Out, "\n%s\n", issue.Description); err != nil {
			return err
		}
	}
	return nil
}

func fieldPlainValue(field domain.IssueField) string {
	if v, ok := field.Value.(domain.DateValue); ok {
		if field.Kind == domain.FieldDate {
			return v.Value.Format("2006-01-02")
		}
		if field.Kind == domain.FieldDateTime {
			return v.Value.Format(time.RFC3339)
		}
	}
	return plainValue(field.Value)
}

func friendlyType(f domain.IssueField) string {
	if f.Cardinality == domain.CardinalityMulti {
		return string(f.Kind) + "[]"
	}
	return string(f.Kind)
}

func plainValue(value domain.FieldValue) string {
	switch v := value.(type) {
	case nil, domain.EmptyValue:
		return ""
	case domain.StringValue:
		return v.Value
	case domain.IntegerValue:
		return fmt.Sprintf("%d", v.Value)
	case domain.FloatValue:
		return fmt.Sprintf("%g", v.Value)
	case domain.DateValue:
		return v.Value.Format(time.RFC3339)
	case domain.PeriodValue:
		h := v.Minutes / 60
		m := v.Minutes % 60
		if h > 0 && m > 0 {
			return fmt.Sprintf("%dh%dm", h, m)
		}
		if h > 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dm", m)
	case domain.TextValue:
		return v.Value
	case domain.EntityValue:
		return v.Name
	case domain.UserValue:
		if v.FullName != "" {
			return v.FullName
		}
		return v.Login
	case domain.StateTransitionValue:
		return v.Presentation
	case domain.MultiValue:
		parts := make([]string, 0, len(v.Values))
		for _, item := range v.Values {
			parts = append(parts, plainValue(item))
		}
		return strings.Join(parts, ", ")
	case domain.UnknownValue:
		return "<unsupported:" + v.Type + ">"
	default:
		return fmt.Sprint(v)
	}
}
