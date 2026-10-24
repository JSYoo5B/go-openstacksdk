package conntrackhelpers

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection over the conntrack helpers of one fixed
// router, bound to direct GET routers/{router_id}/conntrack_helpers/{id} for
// the shared wait engine, such as network.WaitForDelete. Only resource.ID
// references are supported. The record view keeps attributes as raw JSON keyed
// by Python attribute name; resource.WithStatusAttribute selects one of them
// by exact name and requires a JSON string value.
func (a *API) Resources(routerID string) *resource.Collection[Record] {
	return resource.NewCollection(resource.Adapter[Record]{
		WaitAttributes: func(v *Record) map[string]json.RawMessage { return resource.RawResourceView(v.Resource) },
		Kind:           kind,
		Get: func(ctx context.Context, id string) (*Record, error) {
			return a.Get(ctx, routerID, id)
		},
		ID: func(v *Record) string { return v.Text("id") },
	})
}
