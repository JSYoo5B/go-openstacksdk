package portchains

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET sfc/port_chains/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. PortChain has no status attribute; status waits require resource.WithStatusAttribute with an exported string field.
func (a *API) Resources() *resource.Collection[PortChain] {
	return resource.NewCollection(resource.Adapter[PortChain]{
		Kind: "portchains",
		Get:  func(ctx context.Context, id string) (*PortChain, error) { return a.Get(ctx, id) },
		ID:   func(v *PortChain) string { return v.ID },
	})
}
