package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func boardField(boards ...domain.Board) domain.IssueField {
	values := make([]domain.FieldValue, 0, len(boards))
	for _, board := range boards {
		values = append(values, domain.EntityValue{ID: board.ID, Name: board.Name})
	}
	return domain.IssueField{Name: "Board", Kind: domain.FieldBoard, Cardinality: domain.CardinalityMulti, Value: domain.MultiValue{Values: values}}
}

func TestResolveBoardChangeResolvesEligibleValuesAndDiffs(t *testing.T) {
	fake := serviceFixture()
	fake.issue.Fields = []domain.IssueField{boardField(domain.Board{ID: "old", Name: "Legacy"})}
	fake.boards = []domain.Board{
		{ID: "new", Name: "Platform", ProjectIDs: []string{"0-1"}},
		{ID: "other", Name: "Elsewhere", ProjectIDs: []string{"other-project"}},
	}
	service := NewService(fake, nil, fake, nil, fake, fake, fake, fake, fake, fake, nil)

	change, err := service.resolveBoardChange(context.Background(), fake.issue, []string{"new", "platform", "new"}, false)
	require.NoError(t, err)
	assert.Equal(t, []domain.Board{{ID: "new", Name: "Platform", ProjectIDs: []string{"0-1"}}}, change.Add)
	assert.Equal(t, []domain.Board{{ID: "old", Name: "Legacy"}}, change.Remove)

	_, err = service.resolveBoardChange(context.Background(), fake.issue, []string{"Elsewhere"}, false)
	assert.Equal(t, ErrorNotFound, KindOf(err))
}

func TestResolveBoardChangeRejectsUnsafeAndNoneCombinations(t *testing.T) {
	fake := serviceFixture()
	fake.issue.Fields = []domain.IssueField{boardField(domain.Board{ID: "a", Name: "Duplicate"})}
	fake.boards = []domain.Board{{ID: "b", Name: "Duplicate", ProjectIDs: []string{"0-1"}}}
	service := NewService(fake, nil, fake, nil, fake, fake, fake, fake, fake, fake, nil)

	_, err := service.resolveBoardChange(context.Background(), fake.issue, []string{"@none", "Duplicate"}, false)
	assert.Equal(t, ErrorValidation, KindOf(err))
	_, err = service.resolveBoardChange(context.Background(), fake.issue, []string{"b"}, false)
	assert.Equal(t, ErrorAmbiguous, KindOf(err))
}

func TestBoardSetNoopAndEditOrder(t *testing.T) {
	fake := serviceFixture()
	fake.issue.Fields = []domain.IssueField{boardField(domain.Board{ID: "old", Name: "Legacy"})}
	fake.boards = []domain.Board{{ID: "new", Name: "Platform", ProjectIDs: []string{"0-1"}}}
	service := NewService(fake, nil, fake, nil, fake, fake, fake, fake, fake, fake, nil)

	_, err := service.SetField(context.Background(), "TT-1", "Board", []string{"old"})
	require.NoError(t, err)
	assert.Zero(t, fake.boardValidates)
	assert.Zero(t, fake.boardApplies)

	summary := "New"
	_, err = service.EditIssue(context.Background(), "TT-1", EditRequest{Summary: &summary, Fields: []FieldInput{{Name: "Board", Value: "new"}}})
	require.NoError(t, err)
	assert.Equal(t, 1, fake.boardValidates)
	assert.Equal(t, 1, fake.boardApplies)
	assert.Equal(t, 1, fake.updateCalls)
	assert.Equal(t, []domain.Board{{ID: "new", Name: "Platform", ProjectIDs: []string{"0-1"}}}, fake.lastAdd)
	assert.Equal(t, []domain.Board{{ID: "old", Name: "Legacy"}}, fake.lastRemove)
}

func TestEditIssueBoardResolutionPreventsNormalMutation(t *testing.T) {
	fake := serviceFixture()
	fake.boards = []domain.Board{{ID: "new", Name: "Platform", ProjectIDs: []string{"0-1"}}}
	service := NewService(fake, nil, fake, nil, fake, fake, fake, fake, fake, fake, nil)
	summary := "New"
	_, err := service.EditIssue(context.Background(), "TT-1", EditRequest{Summary: &summary, Fields: []FieldInput{{Name: "Board", Value: "missing"}}})
	require.Error(t, err)
	assert.Zero(t, fake.updateCalls)
	assert.Zero(t, fake.boardApplies)
}
