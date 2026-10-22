package serviceprofiles

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET service_profiles/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. ServiceProfile has no status attribute; status waits require resource.WithStatusAttribute with an exported string field such as Driver.
func (a *API) Resources() *resource.Collection[ServiceProfile] {
	return resource.NewCollection(resource.Adapter[ServiceProfile]{
		Kind: "serviceprofiles",
		Get:  func(ctx context.Context, id string) (*ServiceProfile, error) { return a.Get(ctx, id) },
		ID:   func(v *ServiceProfile) string { return v.ID },
	})
}
