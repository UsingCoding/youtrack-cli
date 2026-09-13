package youtrack

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack/dto"
)

func (c *Client) listIssueBoards(ctx context.Context, ref domain.IssueRef) ([]domain.Board, error) {
	items, err := paginate(ctx, func(ctx context.Context, skip, top int) ([]dto.Sprint, error) {
		var page []dto.Sprint
		q := url.Values{"fields": []string{issueSprintFields}, "$skip": []string{itoa(skip)}, "$top": []string{itoa(top)}}
		err := c.doJSON(ctx, http.MethodGet, "/api/issues/"+string(ref)+"/sprints", q, nil, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(items))
	boards := make([]domain.Board, 0, len(items))
	for _, sprint := range items {
		if sprint.Agile == nil || sprint.Agile.ID == "" {
			continue
		}
		if _, ok := seen[sprint.Agile.ID]; ok {
			continue
		}
		seen[sprint.Agile.ID] = struct{}{}
		boards = append(boards, domain.Board{ID: sprint.Agile.ID, Name: sprint.Agile.Name})
	}
	return boards, nil
}

func boardIssueField(boards []domain.Board) domain.IssueField {
	values := make([]domain.FieldValue, 0, len(boards))
	for _, board := range boards {
		values = append(values, domain.EntityValue{ID: board.ID, Name: board.Name})
	}
	return domain.IssueField{ID: "", Name: "Board", Kind: domain.FieldBoard, Cardinality: domain.CardinalityMulti, Value: domain.MultiValue{Values: values}}
}

func (c *Client) ListBoards(ctx context.Context) ([]domain.Board, error) {
	items, err := paginate(ctx, func(ctx context.Context, skip, top int) ([]dto.Agile, error) {
		var page []dto.Agile
		q := url.Values{"fields": []string{boardFields}, "$skip": []string{itoa(skip)}, "$top": []string{itoa(top)}}
		err := c.doJSON(ctx, http.MethodGet, "/api/agiles", q, nil, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}
	boards := make([]domain.Board, 0, len(items))
	for _, item := range items {
		projectIDs := make([]string, 0, len(item.Projects))
		for _, project := range item.Projects {
			projectIDs = append(projectIDs, project.ID)
		}
		boards = append(boards, domain.Board{ID: item.ID, Name: item.Name, ProjectIDs: projectIDs})
	}
	return boards, nil
}

func boardCommandQuery(add, remove []domain.Board) string {
	clauses := make([]string, 0, len(add)+len(remove))
	for _, board := range remove {
		clauses = append(clauses, "remove Board "+board.Name)
	}
	for _, board := range add {
		clauses = append(clauses, "add Board "+board.Name)
	}
	return strings.Join(clauses, " ")
}

func commandPayload(issueID string, add, remove []domain.Board) map[string]any {
	return map[string]any{"query": boardCommandQuery(add, remove), "issues": []map[string]string{{"id": issueID}}}
}

func (c *Client) ValidateIssueBoardChange(ctx context.Context, issueID string, add, remove []domain.Board) error {
	if len(add)+len(remove) == 0 {
		return nil
	}
	var result dto.CommandList
	q := url.Values{"fields": []string{commandValidationFields}}
	if err := c.doJSON(ctx, http.MethodPost, "/api/commands/assist", q, commandPayload(issueID, add, remove), &result); err != nil {
		return err
	}
	if len(result.Commands) != len(add)+len(remove) {
		return app.Validationf("invalid Board change: YouTrack did not parse the requested operations")
	}
	for _, command := range result.Commands {
		if command.Error || command.Delete {
			if command.Description != "" {
				return app.Validationf("invalid Board change: %s", command.Description)
			}
			return app.Validationf("invalid Board change: YouTrack did not parse the requested operations")
		}
	}
	return nil
}

func (c *Client) ApplyIssueBoardChange(ctx context.Context, issueID string, add, remove []domain.Board) error {
	if len(add)+len(remove) == 0 {
		return nil
	}
	return c.doJSON(ctx, http.MethodPost, "/api/commands", nil, commandPayload(issueID, add, remove), nil)
}
