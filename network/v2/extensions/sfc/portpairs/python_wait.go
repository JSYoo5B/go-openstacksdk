package portpairs

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET sfc/port_pairs/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. PortPair has no status attribute; status waits require resource.WithStatusAttribute with an exported string field.
func (a *API) Resources() *resource.Collection[PortPair] {
	return resource.NewCollection(resource.Adapter[PortPair]{
		Kind: "portpairs",
		Get:  func(ctx context.Context, id string) (*PortPair, error) { return a.Get(ctx, id) },
		ID:   func(v *PortPair) string { return v.ID },
	})
}
