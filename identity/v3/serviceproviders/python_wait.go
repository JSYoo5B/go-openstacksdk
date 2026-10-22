package serviceproviders

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET
// OS-FEDERATION/service_providers/{id} for the shared wait engine, such as
// identity.WaitForStatus with resource.WithStatusAttribute and
// identity.WaitForDelete. Only resource.ID references are supported; listing,
// lookup and deletion stay on API. ServiceProvider has no status attribute.
func (a *API) Resources() *resource.Collection[ServiceProvider] {
	return resource.NewCollection(resource.Adapter[ServiceProvider]{
		Kind: "serviceproviders",
		Get:  func(ctx context.Context, id string) (*ServiceProvider, error) { return a.Get(ctx, id) },
		ID:   func(v *ServiceProvider) string { return v.ID },
	})
}
