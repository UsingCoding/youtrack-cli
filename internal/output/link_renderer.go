package output

import (
	"fmt"
	"text/tabwriter"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func (r *Renderer) LinkTypes(items []domain.LinkType) error {
	switch r.Format {
	case FormatJSON:
		return r.json(linkTypesJSON(items))
	case FormatPlain:
		for _, item := range items {
			if _, err := fmt.Fprintln(r.Out, item.ID); err != nil {
				return err
			}
		}
		return nil
	default:
		w := tabwriter.NewWriter(r.Out, 0, 4, 2, ' ', 0)
		if _, err := fmt.Fprintln(w, "ID\tNAME\tDIRECTION\tAGGREGATION"); err != nil {
			return err
		}
		for _, item := range items {
			direction := item.Outward
			if item.Directed {
				direction += " / " + item.Inward
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%t\n", item.ID, item.Name, direction, item.Aggregation); err != nil {
				return err
			}
		}
		return w.Flush()
	}
}

func (r *Renderer) IssueLinks(item domain.IssueLinkList) error {
	switch r.Format {
	case FormatJSON:
		return r.json(issueLinkListJSON(item))
	case FormatPlain:
		return r.IssueSummaries(item.Issues)
	default:
		if _, err := fmt.Fprintf(r.Out, "Relation: %s (%s)\n", item.Relation.Label, item.Relation.Direction); err != nil {
			return err
		}
		return r.IssueSummaries(item.Issues)
	}
}

func (r *Renderer) IssueLinkChange(item domain.IssueLinkChange) error {
	switch r.Format {
	case FormatJSON:
		return r.json(issueLinkChangeJSON(item))
	case FormatPlain:
		return nil
	default:
		if item.Changed {
			if item.Present {
				_, err := fmt.Fprintf(r.Out, "Added link: %s %s %s\n", item.IssueID, item.Relation.Label, item.TargetIssueID)
				return err
			}
			_, err := fmt.Fprintf(r.Out, "Removed link: %s %s %s\n", item.IssueID, item.Relation.Label, item.TargetIssueID)
			return err
		}
		if item.Present {
			_, err := fmt.Fprintf(r.Out, "Link already exists: %s %s %s\n", item.IssueID, item.Relation.Label, item.TargetIssueID)
			return err
		}
		_, err := fmt.Fprintf(r.Out, "Link already absent: %s %s %s\n", item.IssueID, item.Relation.Label, item.TargetIssueID)
		return err
	}
}
