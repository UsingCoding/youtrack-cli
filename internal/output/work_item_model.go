package output

import (
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type WorkItemJSON struct {
	EntityID        string            `json:"entityId"`
	Date            string            `json:"date"`
	DurationMinutes int64             `json:"durationMinutes"`
	Text            *string           `json:"text"`
	Author          *WorkItemUserJSON `json:"author"`
	Creator         *WorkItemUserJSON `json:"creator"`
	Type            *WorkItemTypeJSON `json:"type"`
	Created         time.Time         `json:"created"`
	Updated         *time.Time        `json:"updated"`
}

type WorkItemUserJSON struct {
	EntityID string `json:"entityId"`
	Login    string `json:"login"`
	Name     string `json:"name"`
}

type WorkItemTypeJSON struct {
	EntityID string `json:"entityId"`
	Name     string `json:"name"`
}

type WorkItemRemovalJSON struct {
	EntityID string `json:"entityId"`
	Removed  bool   `json:"removed"`
}

func workItemJSON(item domain.WorkItem) WorkItemJSON {
	out := WorkItemJSON{
		EntityID: item.ID, Date: item.Date.UTC().Format("2006-01-02"), DurationMinutes: item.DurationMinutes,
		Text: item.Text, Created: item.Created.UTC(),
	}
	if item.Author != nil {
		out.Author = &WorkItemUserJSON{EntityID: item.Author.ID, Login: item.Author.Login, Name: item.Author.FullName}
	}
	if item.Creator != nil {
		out.Creator = &WorkItemUserJSON{EntityID: item.Creator.ID, Login: item.Creator.Login, Name: item.Creator.FullName}
	}
	if item.Type != nil {
		out.Type = &WorkItemTypeJSON{EntityID: item.Type.ID, Name: item.Type.Name}
	}
	if item.Updated != nil {
		updated := item.Updated.UTC()
		out.Updated = &updated
	}
	return out
}

func workItemsJSON(items []domain.WorkItem) []WorkItemJSON {
	out := make([]WorkItemJSON, 0, len(items))
	for _, item := range items {
		out = append(out, workItemJSON(item))
	}
	return out
}

func workItemTypesJSON(items []domain.WorkItemType) []WorkItemTypeJSON {
	out := make([]WorkItemTypeJSON, 0, len(items))
	for _, item := range items {
		out = append(out, WorkItemTypeJSON{EntityID: item.ID, Name: item.Name})
	}
	return out
}
