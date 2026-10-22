package portpairgroups

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET sfc/port_pair_groups/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. PortPairGroup has no status attribute; status waits require resource.WithStatusAttribute with an exported string field.
func (a *API) Resources() *resource.Collection[PortPairGroup] {
	return resource.NewCollection(resource.Adapter[PortPairGroup]{
		Kind: "portpairgroups",
		Get:  func(ctx context.Context, id string) (*PortPairGroup, error) { return a.Get(ctx, id) },
		ID:   func(v *PortPairGroup) string { return v.ID },
	})
}
