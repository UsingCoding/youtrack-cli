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

func (c *Client) ListWorkItems(ctx context.Context, issue domain.IssueRef, page app.Page) ([]domain.WorkItem, error) {
	var data []dto.WorkItem
	params := url.Values{
		"fields": []string{workItemFields},
		"$skip":  []string{itoa(page.Offset)},
		"$top":   []string{itoa(page.Limit)},
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/issues/"+string(issue)+"/timeTracking/workItems", params, nil, &data); err != nil {
		return nil, err
	}
	items := make([]domain.WorkItem, 0, len(data))
	for _, item := range data {
		items = append(items, mapWorkItem(item))
	}
	return items, nil
}

func (c *Client) GetWorkItem(ctx context.Context, issue domain.IssueRef, itemID string) (domain.WorkItem, error) {
	var data dto.WorkItem
	params := url.Values{"fields": []string{workItemFields}}
	path := "/api/issues/" + string(issue) + "/timeTracking/workItems/" + itemID
	if err := c.doJSON(ctx, http.MethodGet, path, params, nil, &data); err != nil {
		return domain.WorkItem{}, err
	}
	return mapWorkItem(data), nil
}

func (c *Client) CreateWorkItem(ctx context.Context, issue domain.IssueRef, create app.WorkItemCreate) (domain.WorkItem, error) {
	date, err := serializeWorkItemDate(create.Date)
	if err != nil {
		return domain.WorkItem{}, err
	}
	payload := map[string]any{
		"date":     date,
		"duration": map[string]int64{"minutes": create.DurationMinutes},
	}
	if create.Text != nil {
		payload["text"] = *create.Text
	}
	if create.Type != nil {
		payload["type"] = map[string]string{"id": create.Type.ID}
	}
	if create.Author != nil {
		payload["author"] = map[string]string{"id": create.Author.ID}
	}
	var data dto.WorkItem
	path := "/api/issues/" + string(issue) + "/timeTracking/workItems"
	if err := c.doJSON(ctx, http.MethodPost, path, url.Values{"fields": []string{workItemFields}}, payload, &data); err != nil {
		return domain.WorkItem{}, err
	}
	return mapWorkItem(data), nil
}

func (c *Client) UpdateWorkItem(ctx context.Context, issue domain.IssueRef, itemID string, patch app.WorkItemPatch) (domain.WorkItem, error) {
	payload := map[string]any{}
	if patch.DurationMinutes != nil {
		payload["duration"] = map[string]int64{"minutes": *patch.DurationMinutes}
	}
	if patch.Date != nil {
		date, err := serializeWorkItemDate(*patch.Date)
		if err != nil {
			return domain.WorkItem{}, err
		}
		payload["date"] = date
	}
	if patch.Text != nil {
		payload["text"] = *patch.Text
	}
	if patch.Type.Set {
		if patch.Type.Value == nil {
			payload["type"] = nil
		} else {
			payload["type"] = map[string]string{"id": patch.Type.Value.ID}
		}
	}
	if patch.Author != nil {
		payload["author"] = map[string]string{"id": patch.Author.ID}
	}
	var data dto.WorkItem
	path := "/api/issues/" + string(issue) + "/timeTracking/workItems/" + itemID
	if err := c.doJSON(ctx, http.MethodPost, path, url.Values{"fields": []string{workItemFields}}, payload, &data); err != nil {
		return domain.WorkItem{}, err
	}
	return mapWorkItem(data), nil
}

func (c *Client) DeleteWorkItem(ctx context.Context, issue domain.IssueRef, itemID string) error {
	path := "/api/issues/" + string(issue) + "/timeTracking/workItems/" + itemID
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *Client) TimeTrackingEnabled(ctx context.Context, projectID string) (bool, error) {
	var data dto.TimeTrackingSettings
	params := url.Values{"fields": []string{timeTrackingSettingsFields}}
	path := "/api/admin/projects/" + projectID + "/timeTrackingSettings"
	if err := c.doJSON(ctx, http.MethodGet, path, params, nil, &data); err != nil {
		return false, err
	}
	return data.Enabled, nil
}

func (c *Client) ListWorkItemTypes(ctx context.Context, projectID string, page app.Page) ([]domain.WorkItemType, error) {
	var data []dto.WorkItemType
	params := url.Values{
		"fields": []string{workItemTypeFields},
		"$skip":  []string{itoa(page.Offset)},
		"$top":   []string{itoa(page.Limit)},
	}
	path := "/api/admin/projects/" + projectID + "/timeTrackingSettings/workItemTypes"
	if err := c.doJSON(ctx, http.MethodGet, path, params, nil, &data); err != nil {
		return nil, err
	}
	items := make([]domain.WorkItemType, 0, len(data))
	for _, item := range data {
		items = append(items, domain.WorkItemType{ID: item.ID, Name: item.Name})
	}
	return items, nil
}

func mapWorkItem(item dto.WorkItem) domain.WorkItem {
	out := domain.WorkItem{
		ID: item.ID, Date: time.UnixMilli(item.Date).UTC(), DurationMinutes: item.Duration.Minutes,
		Text: item.Text, Created: time.UnixMilli(item.Created).UTC(),
	}
	if item.Author != nil {
		out.Author = &domain.User{ID: item.Author.ID, Login: item.Author.Login, FullName: item.Author.FullName}
	}
	if item.Creator != nil {
		out.Creator = &domain.User{ID: item.Creator.ID, Login: item.Creator.Login, FullName: item.Creator.FullName}
	}
	if item.Type != nil {
		out.Type = &domain.WorkItemType{ID: item.Type.ID, Name: item.Type.Name}
	}
	if item.Updated != nil {
		updated := time.UnixMilli(*item.Updated).UTC()
		out.Updated = &updated
	}
	return out
}

func serializeWorkItemDate(date time.Time) (int64, error) {
	if date.Location() != time.UTC || date.Hour() != 0 || date.Minute() != 0 || date.Second() != 0 || date.Nanosecond() != 0 {
		return 0, app.Validationf("work item date must be UTC midnight")
	}
	return date.UnixMilli(), nil
}
