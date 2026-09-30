package resource

import (
	"context"
	"iter"

	"github.com/gophercloud/gophercloud/v2/pagination"
)

// Stream extracts resources from each page while preserving cancellation and
// stopping pagination when the consumer breaks out of the iterator.
func Stream[T any](ctx context.Context, pager pagination.Pager, extract func(pagination.Page) ([]T, error)) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		stopped := false
		err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
			values, err := extract(page)
			if err != nil {
				return false, err
			}
			for i := range values {
				if !yield(&values[i], nil) {
					stopped = true
					return false, nil
				}
			}
			return true, nil
		})
		if err != nil && !stopped {
			yield(nil, err)
		}
	}
}

// StreamValues handles typed page responses such as address maps or allocation
// candidates, which are not naturally represented as slices of resource objects.
func StreamValues[T any](ctx context.Context, pager pagination.Pager, extract func(pagination.Page) (T, error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		stopped := false
		err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
			value, err := extract(page)
			if err != nil {
				return false, err
			}
			if !yield(value, nil) {
				stopped = true
				return false, nil
			}
			return true, nil
		})
		if err != nil && !stopped {
			var zero T
			yield(zero, err)
		}
	}
}

// Pages is reserved for APIs whose upstream page has no typed extractor.
func Pages(ctx context.Context, pager pagination.Pager) iter.Seq2[pagination.Page, error] {
	return func(yield func(pagination.Page, error) bool) {
		stopped := false
		err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
			if !yield(page, nil) {
				stopped = true
				return false, nil
			}
			return true, nil
		})
		if err != nil && !stopped {
			yield(nil, err)
		}
	}
}
