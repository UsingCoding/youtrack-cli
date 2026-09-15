package output

import (
	"fmt"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func (r *Renderer) SavedSearch(search domain.SavedSearch, issues []domain.IssueSummary) error {
	switch r.Format {
	case FormatJSON:
		out := SavedSearchJSON{
			EntityID: search.ID,
			Name:     search.Name,
			Query:    search.Query,
			Issues:   issueSummariesJSON(issues),
		}
		if search.Owner != nil {
			out.Owner = &UserJSON{EntityID: search.Owner.ID, Login: search.Owner.Login, Name: search.Owner.FullName}
		}
		return r.json(out)
	case FormatPlain:
		return r.IssueSummaries(issues)
	default:
		if _, err := fmt.Fprintf(r.Out, "Name: %s\nQuery: %s\n", search.Name, search.Query); err != nil {
			return err
		}
		if search.Owner != nil {
			if _, err := fmt.Fprintf(r.Out, "Owner: %s (%s)\n", search.Owner.FullName, search.Owner.Login); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(r.Out); err != nil {
			return err
		}
		return r.IssueSummaries(issues)
	}
}
