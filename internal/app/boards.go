package app

import (
	"context"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type boardChange struct {
	Add    []domain.Board
	Remove []domain.Board
}

func (c boardChange) Empty() bool { return len(c.Add) == 0 && len(c.Remove) == 0 }

func isBoardRef(ref string) bool { return strings.EqualFold(strings.TrimSpace(ref), "Board") }

func issueBoards(issue domain.Issue) []domain.Board {
	for _, field := range issue.Fields {
		if field.Kind != domain.FieldBoard {
			continue
		}
		value, ok := field.Value.(domain.MultiValue)
		if !ok {
			return nil
		}
		boards := make([]domain.Board, 0, len(value.Values))
		for _, item := range value.Values {
			if entity, ok := item.(domain.EntityValue); ok && entity.ID != "" {
				boards = append(boards, domain.Board{ID: entity.ID, Name: entity.Name})
			}
		}
		return boards
	}
	return nil
}

func (s *Service) resolveBoardChange(ctx context.Context, issue domain.Issue, values []string, clear bool) (boardChange, error) {
	current := issueBoards(issue)
	if !clear {
		for _, value := range values {
			if strings.TrimSpace(value) == "@none" && len(values) != 1 {
				return boardChange{}, Validationf("field %q cannot combine @none with board values", "Board")
			}
		}
	}
	if clear || (len(values) == 1 && strings.TrimSpace(values[0]) == "@none") {
		return boardDiff(current, nil), nil
	}

	available, err := s.boards.ListBoards(ctx)
	if err != nil {
		return boardChange{}, err
	}
	candidates := append([]domain.Board(nil), current...)
	known := make(map[string]struct{}, len(candidates))
	for _, board := range candidates {
		known[board.ID] = struct{}{}
	}
	for _, board := range available {
		if !containsID(board.ProjectIDs, issue.Project.ID) {
			continue
		}
		if _, ok := known[board.ID]; ok {
			continue
		}
		known[board.ID] = struct{}{}
		candidates = append(candidates, board)
	}

	desired := make([]domain.Board, 0, len(values))
	selected := make(map[string]struct{}, len(values))
	for _, value := range values {
		board, err := resolveBoard(candidates, strings.TrimSpace(value))
		if err != nil {
			return boardChange{}, err
		}
		if _, ok := selected[board.ID]; ok {
			continue
		}
		selected[board.ID] = struct{}{}
		desired = append(desired, board)
	}
	change := boardDiff(current, desired)
	for _, changed := range append(append([]domain.Board(nil), change.Remove...), change.Add...) {
		for _, candidate := range candidates {
			if candidate.ID != changed.ID && candidate.Name == changed.Name {
				return boardChange{}, Ambiguousf("value %q is ambiguous for field %q", changed.Name, "Board")
			}
		}
	}
	return change, nil
}

func containsID(ids []string, id string) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func resolveBoard(boards []domain.Board, input string) (domain.Board, error) {
	for _, board := range boards {
		if board.ID == input {
			return board, nil
		}
	}
	var exact []domain.Board
	for _, board := range boards {
		if board.Name == input {
			exact = append(exact, board)
		}
	}
	if len(exact) == 1 {
		return exact[0], nil
	}
	if len(exact) > 1 {
		return domain.Board{}, Ambiguousf("value %q is ambiguous for field %q", input, "Board")
	}
	var folded []domain.Board
	for _, board := range boards {
		if strings.EqualFold(board.Name, input) {
			folded = append(folded, board)
		}
	}
	if len(folded) == 1 {
		return folded[0], nil
	}
	if len(folded) > 1 {
		return domain.Board{}, Ambiguousf("value %q is ambiguous for field %q", input, "Board")
	}
	return domain.Board{}, NotFoundf("value %q was not found for field %q", input, "Board")
}

func boardDiff(current, desired []domain.Board) boardChange {
	desiredIDs := make(map[string]struct{}, len(desired))
	for _, board := range desired {
		desiredIDs[board.ID] = struct{}{}
	}
	currentIDs := make(map[string]struct{}, len(current))
	change := boardChange{}
	for _, board := range current {
		currentIDs[board.ID] = struct{}{}
		if _, ok := desiredIDs[board.ID]; !ok {
			change.Remove = append(change.Remove, board)
		}
	}
	for _, board := range desired {
		if _, ok := currentIDs[board.ID]; !ok {
			change.Add = append(change.Add, board)
		}
	}
	return change
}
