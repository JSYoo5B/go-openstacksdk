package resource

import (
	"context"
	"iter"
)

// ListControl governs iteration before local name and status filters. Zero
// MaxItems is unlimited; SinglePage prevents requesting a continuation page.
// Service bindings determine whether a local cap also supplies a wire limit.
type ListControl struct {
	MaxItems   int
	SinglePage bool
}

func capIterator[T any](ctx context.Context, source iter.Seq2[*T, error], maximum int) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		count := 0
		for value, err := range source {
			if !yield(value, err) || err != nil {
				return
			}
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				return
			}
			count++
			if maximum > 0 && count >= maximum {
				return
			}
		}
	}
}
