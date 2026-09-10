package youtrack

import "context"

const pageSize = 100

func paginate[T any](ctx context.Context, fetch func(context.Context, int, int) ([]T, error)) ([]T, error) {
	var result []T
	for skip := 0; ; {
		page, err := fetch(ctx, skip, pageSize)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return result, nil
		}
		result = append(result, page...)
		skip += len(page)
	}
}
