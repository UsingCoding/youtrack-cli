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

type commentTextRequest struct {
	Text string `json:"text"`
}

type commentRemoveRequest struct {
	Deleted bool `json:"deleted"`
}

func (c *Client) ListComments(ctx context.Context, issue domain.IssueRef, page app.Page) ([]domain.Comment, error) {
	data := make([]dto.Comment, 0)
	params := url.Values{
		"fields": []string{commentFields},
		"$skip":  []string{itoa(page.Offset)},
		"$top":   []string{itoa(page.Limit)},
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/issues/"+string(issue)+"/comments", params, nil, &data); err != nil {
		return nil, err
	}
	items := make([]domain.Comment, 0, len(data))
	for _, item := range data {
		items = append(items, mapComment(item))
	}
	return items, nil
}

func (c *Client) CreateComment(ctx context.Context, issue domain.IssueRef, text string) (domain.Comment, error) {
	var data dto.Comment
	path := "/api/issues/" + string(issue) + "/comments"
	if err := c.doJSON(ctx, http.MethodPost, path, url.Values{"fields": []string{commentFields}}, commentTextRequest{Text: text}, &data); err != nil {
		return domain.Comment{}, err
	}
	return mapComment(data), nil
}

func (c *Client) EditComment(ctx context.Context, issue domain.IssueRef, commentID, text string) (domain.Comment, error) {
	var data dto.Comment
	path := "/api/issues/" + string(issue) + "/comments/" + commentID
	if err := c.doJSON(ctx, http.MethodPost, path, url.Values{"fields": []string{commentFields}}, commentTextRequest{Text: text}, &data); err != nil {
		return domain.Comment{}, err
	}
	return mapComment(data), nil
}

func (c *Client) SoftRemoveComment(ctx context.Context, issue domain.IssueRef, commentID string) error {
	path := "/api/issues/" + string(issue) + "/comments/" + commentID
	return c.doJSON(ctx, http.MethodPost, path, nil, commentRemoveRequest{Deleted: true}, nil)
}

func mapComment(item dto.Comment) domain.Comment {
	comment := domain.Comment{ID: item.ID, Text: item.Text, Created: time.UnixMilli(item.Created), Deleted: item.Deleted}
	if item.Author != nil {
		comment.Author = &domain.User{ID: item.Author.ID, Login: item.Author.Login, FullName: item.Author.FullName}
	}
	if item.Updated != nil {
		updated := time.UnixMilli(*item.Updated)
		comment.Updated = &updated
	}
	return comment
}
