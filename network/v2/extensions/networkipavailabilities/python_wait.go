package networkipavailabilities

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET
// network-ip-availabilities/{network_id} for the shared wait engine, such as
// network.WaitForStatus with resource.WithStatusAttribute and
// network.WaitForDelete. Only resource.ID references are supported.
func (a *API) Resources() *resource.Collection[NetworkIPAvailability] {
	return resource.NewCollection(resource.Adapter[NetworkIPAvailability]{
		Kind: "networkipavailabilities",
		Get: func(ctx context.Context, id string) (*NetworkIPAvailability, error) {
			return a.Get(ctx, id)
		},
		ID: func(v *NetworkIPAvailability) string { return v.NetworkID },
	})
}
