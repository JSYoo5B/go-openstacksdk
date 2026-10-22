package ndpproxies

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET ndp_proxies/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. Record keeps attributes as raw JSON, so it exposes no exported string attribute for status waits.
func (a *API) Resources() *resource.Collection[Record] {
	return resource.NewCollection(resource.Adapter[Record]{
		Kind: kind,
		Get:  func(ctx context.Context, id string) (*Record, error) { return a.Get(ctx, id) },
		ID:   func(v *Record) string { return v.Text("id") },
	})
}
