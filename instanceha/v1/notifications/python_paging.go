package notifications

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
func (a *API) ListMarkerFallback(ctx context.Context, options ...ListOption) iter.Seq2[*Notification, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Notification, error) bool) {
		query, err := prepareList(options...)
		if err != nil {
			yield(nil, request.Wrap("ListMarkerFallback", "notifications", err))
			return
		}
		for value, err := range masakari.ListMarkerFallback(ctx, a.spec, query, "ListMarkerFallback", "notifications") {
			if !yield(value, err) {
				return
			}
		}
	}
}

// AllMarkerFallback collects ListMarkerFallback into a non-nil slice.
func (a *API) AllMarkerFallback(ctx context.Context, options ...ListOption) ([]*Notification, error) {
	values := make([]*Notification, 0)
	for value, err := range a.ListMarkerFallback(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
