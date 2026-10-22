package conntrackhelpers

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection over the conntrack helpers of one fixed
// router, bound to direct GET routers/{router_id}/conntrack_helpers/{id} for
// the shared wait engine, such as network.WaitForDelete. Only resource.ID
// references are supported. Record keeps attributes as raw JSON, so it exposes
// no exported string attribute for status waits.
func (a *API) Resources(routerID string) *resource.Collection[Record] {
	return resource.NewCollection(resource.Adapter[Record]{
		Kind: kind,
		Get: func(ctx context.Context, id string) (*Record, error) {
			return a.Get(ctx, routerID, id)
		},
		ID: func(v *Record) string { return v.Text("id") },
	})
}
