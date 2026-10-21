package hosts

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ListMarkerFallback lists like the pinned Python Resource.list. It follows an
// advertised next link and, only when an explicit limit was sent and the page
// was not empty, requests the next page with marker set to the wire "id" of
// the last row. Prefer List unless that Python behavior is required: the
// server's marker identity (database id or UUID) is not verified, so the
// extra request can be rejected by the controller. Iteration stops on an empty
// page or a repeated page/marker. Options and validation are those of List.
func (s *SegmentScope) ListMarkerFallback(ctx context.Context, options ...ListOption) iter.Seq2[*Host, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Host, error) bool) {
		query, err := prepareList(options...)
		if err != nil {
			yield(nil, request.Wrap("ListMarkerFallback", "hosts", err))
			return
		}
		for value, err := range masakari.ListMarkerFallback(ctx, s.spec, query, "ListMarkerFallback", "hosts") {
			if !yield(value, err) {
				return
			}
		}
	}
}

// AllMarkerFallback collects ListMarkerFallback into a non-nil slice.
func (s *SegmentScope) AllMarkerFallback(ctx context.Context, options ...ListOption) ([]*Host, error) {
	values := make([]*Host, 0)
	for value, err := range s.ListMarkerFallback(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
