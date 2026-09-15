package output

import (
	"fmt"
	"text/tabwriter"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func (r *Renderer) IssueSummaries(items []domain.IssueSummary) error {
	switch r.Format {
	case FormatJSON:
		out := make([]IssueSummaryJSON, 0, len(items))
		for _, item := range items {
			out = append(out, IssueSummaryJSON{
				ID: item.IDReadable, EntityID: item.ID, Summary: item.Summary,
				Project: ProjectJSON{EntityID: item.Project.ID, Name: item.Project.Name, ShortName: item.Project.ShortName},
				Created: item.Created, Updated: item.Updated, Resolved: item.Resolved,
			})
		}
		return r.json(out)
	case FormatPlain:
		for _, item := range items {
			if _, err := fmt.Fprintf(r.Out, "%s\t%s\n", item.IDReadable, item.Summary); err != nil {
				return err
			}
		}
		return nil
	default:
		w := tabwriter.NewWriter(r.Out, 0, 4, 2, ' ', 0)
		if _, err := fmt.Fprintln(w, "ID\tPROJECT\tUPDATED\tSUMMARY"); err != nil {
			return err
		}
		for _, item := range items {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", item.IDReadable, item.Project.ShortName, item.Updated.Format("2006-01-02 15:04"), item.Summary); err != nil {
				return err
			}
		}
		return w.Flush()
	}
}
