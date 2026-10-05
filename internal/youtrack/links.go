package youtrack

import (
	"context"
	"net/http"
	"net/url"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack/dto"
)

func (c *Client) ListLinkTypes(ctx context.Context, page app.Page) ([]domain.LinkType, error) {
	var data []dto.LinkType
	params := url.Values{
		"fields": []string{linkTypeFields},
		"$skip":  []string{itoa(page.Offset)},
		"$top":   []string{itoa(page.Limit)},
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/issueLinkTypes", params, nil, &data); err != nil {
		return nil, err
	}
	items := make([]domain.LinkType, 0, len(data))
	for _, item := range data {
		items = append(items, domain.LinkType{
			ID: item.ID, Name: item.Name, Directed: item.Directed, Aggregation: item.Aggregation,
			Outward: item.SourceToTarget, Inward: item.TargetToSource,
		})
	}
	return items, nil
}

func (c *Client) ListLinkedIssues(ctx context.Context, source domain.IssueRef, relation domain.LinkRelation, page app.Page) ([]domain.IssueSummary, error) {
	id, err := linkID(relation)
	if err != nil {
		return nil, err
	}
	var data []dto.IssueSummary
	params := url.Values{
		"fields": []string{issueSummaryFields},
		"$skip":  []string{itoa(page.Offset)},
		"$top":   []string{itoa(page.Limit)},
	}
	path := "/api/issues/" + string(source) + "/links/" + id + "/issues"
	if err := c.doJSON(ctx, http.MethodGet, path, params, nil, &data); err != nil {
		return nil, err
	}
	items := make([]domain.IssueSummary, 0, len(data))
	for _, item := range data {
		items = append(items, mapIssueSummary(item))
	}
	return items, nil
}

func (c *Client) AddIssueLink(ctx context.Context, source domain.IssueRef, relation domain.LinkRelation, target domain.IssueRef) error {
	id, err := linkID(relation)
	if err != nil {
		return err
	}
	path := "/api/issues/" + string(source) + "/links/" + id + "/issues"
	return c.doJSON(ctx, http.MethodPost, path, nil, map[string]string{"id": string(target)}, nil)
}

func (c *Client) RemoveIssueLink(ctx context.Context, source domain.IssueRef, relation domain.LinkRelation, target domain.IssueRef) error {
	id, err := linkID(relation)
	if err != nil {
		return err
	}
	path := "/api/issues/" + string(source) + "/links/" + id + "/issues/" + string(target)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func linkID(relation domain.LinkRelation) (string, error) {
	if relation.Type.ID == "" {
		return "", app.Validationf("link type ID must not be blank")
	}
	if !relation.Type.Directed {
		if relation.Direction != domain.LinkDirectionUndirected {
			return "", app.Validationf("undirected link type %q has invalid direction", relation.Type.Name)
		}
		return relation.Type.ID, nil
	}
	switch relation.Direction {
	case domain.LinkDirectionOutward:
		return relation.Type.ID + "s", nil
	case domain.LinkDirectionInward:
		return relation.Type.ID + "t", nil
	default:
		return "", app.Validationf("directed link type %q has invalid direction", relation.Type.Name)
	}
}
