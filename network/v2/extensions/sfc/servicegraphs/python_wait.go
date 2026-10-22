package servicegraphs

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET sfc/service_graphs/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. ServiceGraph has no status attribute; status waits require resource.WithStatusAttribute with an exported string field.
func (a *API) Resources() *resource.Collection[ServiceGraph] {
	return resource.NewCollection(resource.Adapter[ServiceGraph]{
		Kind: "servicegraphs",
		Get:  func(ctx context.Context, id string) (*ServiceGraph, error) { return a.Get(ctx, id) },
		ID:   func(v *ServiceGraph) string { return v.ID },
	})
}
