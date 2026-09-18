package output

import (
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type IssueSummaryJSON struct {
	ID       string      `json:"id"`
	EntityID string      `json:"entityId"`
	Summary  string      `json:"summary"`
	Project  ProjectJSON `json:"project"`
	Created  time.Time   `json:"created"`
	Updated  time.Time   `json:"updated"`
	Resolved *time.Time  `json:"resolved"`
}

func issueSummaryJSON(item domain.IssueSummary) IssueSummaryJSON {
	return IssueSummaryJSON{
		ID: item.IDReadable, EntityID: item.ID, Summary: item.Summary,
		Project: ProjectJSON{EntityID: item.Project.ID, Name: item.Project.Name, ShortName: item.Project.ShortName},
		Created: item.Created, Updated: item.Updated, Resolved: item.Resolved,
	}
}

func issueSummariesJSON(items []domain.IssueSummary) []IssueSummaryJSON {
	out := make([]IssueSummaryJSON, 0, len(items))
	for _, item := range items {
		out = append(out, issueSummaryJSON(item))
	}
	return out
}
