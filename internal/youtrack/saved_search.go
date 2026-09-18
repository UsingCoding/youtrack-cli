package youtrack

import (
	"context"
	"net/http"
	"net/url"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack/dto"
)

func (c *Client) GetSavedSearch(ctx context.Context, id string) (domain.SavedSearch, error) {
	var data dto.SavedQuery
	if err := c.doJSON(ctx, http.MethodGet, "/api/savedQueries/"+id, url.Values{"fields": []string{savedQueryFields}}, nil, &data); err != nil {
		return domain.SavedSearch{}, err
	}
	return mapSavedSearch(data), nil
}

func (c *Client) ListSavedSearches(ctx context.Context, page app.Page) ([]domain.SavedSearch, error) {
	data := make([]dto.SavedQuery, 0)
	params := url.Values{
		"fields": []string{savedQueryFields},
		"$skip":  []string{itoa(page.Offset)},
		"$top":   []string{itoa(page.Limit)},
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/savedQueries", params, nil, &data); err != nil {
		return nil, err
	}
	items := make([]domain.SavedSearch, 0, len(data))
	for _, item := range data {
		items = append(items, mapSavedSearch(item))
	}
	return items, nil
}

func mapSavedSearch(item dto.SavedQuery) domain.SavedSearch {
	search := domain.SavedSearch{ID: item.ID, Name: item.Name}
	if item.Query != nil {
		search.Query = *item.Query
	}
	if item.Owner != nil {
		search.Owner = &domain.User{ID: item.Owner.ID, Login: item.Owner.Login, FullName: item.Owner.FullName}
	}
	return search
}
