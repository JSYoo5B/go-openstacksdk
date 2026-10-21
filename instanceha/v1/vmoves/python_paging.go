package vmoves

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
func (s *NotificationScope) ListMarkerFallback(ctx context.Context, options ...ListOption) iter.Seq2[*VMove, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*VMove, error) bool) {
		query, err := prepareList(options...)
		if err != nil {
			yield(nil, request.Wrap("ListMarkerFallback", "vmoves", err))
			return
		}
		for value, err := range masakari.ListMarkerFallback(ctx, s.spec, query, "ListMarkerFallback", "vmoves") {
			if !yield(value, err) {
				return
			}
		}
	}
}

// AllMarkerFallback collects ListMarkerFallback into a non-nil slice.
func (s *NotificationScope) AllMarkerFallback(ctx context.Context, options ...ListOption) ([]*VMove, error) {
	values := make([]*VMove, 0)
	for value, err := range s.ListMarkerFallback(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
