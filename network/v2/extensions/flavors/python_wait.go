package flavors

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET flavors/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. Flavor has no status attribute; status waits require resource.WithStatusAttribute with an exported string field such as ServiceType.
func (a *API) Resources() *resource.Collection[Flavor] {
	return resource.NewCollection(resource.Adapter[Flavor]{
		Kind: "flavors",
		Get:  func(ctx context.Context, id string) (*Flavor, error) { return a.Get(ctx, id) },
		ID:   func(v *Flavor) string { return v.ID },
	})
}
