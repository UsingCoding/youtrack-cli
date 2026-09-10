package youtrack

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaginateDoesNotStopWhenServerReturnsLessThanRequestedTop(t *testing.T) {
	calls := []int{}
	items, err := paginate(context.Background(), func(_ context.Context, skip, top int) ([]int, error) {
		assert.Equal(t, pageSize, top)
		calls = append(calls, skip)
		switch skip {
		case 0:
			page := make([]int, 42)
			for i := range page {
				page[i] = i
			}
			return page, nil
		case 42:
			return []int{42}, nil
		default:
			return nil, nil
		}
	})

	require.NoError(t, err)
	assert.Len(t, items, 43)
	assert.Equal(t, []int{0, 42, 43}, calls)
}
