package youtrack

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack/dto"
)

func (c *Client) SearchIssues(ctx context.Context, query string, page app.Page) ([]domain.IssueSummary, error) {
	data := make([]dto.IssueSummary, 0)
	params := url.Values{
		"query":  []string{query},
		"$skip":  []string{itoa(page.Offset)},
		"$top":   []string{itoa(page.Limit)},
		"fields": []string{issueSummaryFields},
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/issues", params, nil, &data); err != nil {
		return nil, err
	}
	items := make([]domain.IssueSummary, 0, len(data))
	for _, item := range data {
		mapped := domain.IssueSummary{
			ID: item.ID, IDReadable: item.IDReadable, Summary: item.Summary,
			Project: domain.Project{ID: item.Project.ID, Name: item.Project.Name, ShortName: item.Project.ShortName},
			Created: time.UnixMilli(item.Created), Updated: time.UnixMilli(item.Updated),
		}
		if item.Resolved != nil {
			resolved := time.UnixMilli(*item.Resolved)
			mapped.Resolved = &resolved
		}
		items = append(items, mapped)
	}
	return items, nil
}
