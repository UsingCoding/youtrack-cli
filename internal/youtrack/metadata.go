package youtrack

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack/dto"
)

func (c *Client) Me(ctx context.Context) (domain.User, error) {
	var u dto.User
	q := url.Values{"fields": []string{userFields}}
	if err := c.doJSON(ctx, http.MethodGet, "/api/users/me", q, nil, &u); err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: u.ID, Login: u.Login, FullName: u.FullName}, nil
}

func (c *Client) ListUsers(ctx context.Context, page app.Page) ([]domain.User, error) {
	var data []dto.User
	params := url.Values{
		"fields": []string{userFields},
		"$skip":  []string{itoa(page.Offset)},
		"$top":   []string{itoa(page.Limit)},
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/users", params, nil, &data); err != nil {
		return nil, err
	}
	items := make([]domain.User, 0, len(data))
	for _, item := range data {
		items = append(items, domain.User{ID: item.ID, Login: item.Login, FullName: item.FullName})
	}
	return items, nil
}

func (c *Client) SearchTags(ctx context.Context, query string) ([]domain.Tag, error) {
	items, err := paginate(ctx, func(ctx context.Context, skip, top int) ([]dto.Tag, error) {
		var page []dto.Tag
		q := url.Values{"fields": []string{tagFields}, "$skip": []string{itoa(skip)}, "$top": []string{itoa(top)}, "query": []string{query}}
		err := c.doJSON(ctx, http.MethodGet, "/api/tags", q, nil, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Tag, 0, len(items))
	for _, t := range items {
		out = append(out, domain.Tag{ID: t.ID, Name: t.Name})
	}
	return out, nil
}

func (c *Client) SearchGroups(ctx context.Context, query string) ([]domain.Group, error) {
	items, err := paginate(ctx, func(ctx context.Context, skip, top int) ([]dto.Group, error) {
		var page []dto.Group
		q := url.Values{"fields": []string{groupFields}, "$skip": []string{itoa(skip)}, "$top": []string{itoa(top)}, "query": []string{query}}
		err := c.doJSON(ctx, http.MethodGet, "/api/groups", q, nil, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Group, 0, len(items))
	for _, g := range items {
		out = append(out, domain.Group{ID: g.ID, Name: g.Name})
	}
	return out, nil
}

func itoa(v int) string { return strconv.Itoa(v) }
