package identityproviders

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET
// OS-FEDERATION/identity_providers/{id} for the shared wait engine, such as
// identity.WaitForStatus with resource.WithStatusAttribute and
// identity.WaitForDelete. Only resource.ID references are supported; listing,
// lookup and deletion stay on API. IdentityProvider has no status attribute.
func (a *API) Resources() *resource.Collection[IdentityProvider] {
	return resource.NewCollection(resource.Adapter[IdentityProvider]{
		Kind: "identityproviders",
		Get:  func(ctx context.Context, id string) (*IdentityProvider, error) { return a.Get(ctx, id) },
		ID:   func(v *IdentityProvider) string { return v.ID },
	})
}
