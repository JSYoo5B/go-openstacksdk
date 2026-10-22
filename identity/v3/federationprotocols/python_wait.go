package federationprotocols

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection over the protocols of one fixed identity
// provider, bound to direct GET
// OS-FEDERATION/identity_providers/{idp_id}/protocols/{id} for the shared wait
// engine, such as identity.WaitForDelete. Only resource.ID references are
// supported. FederationProtocol has no status attribute.
func (a *API) Resources(identityProviderID string) *resource.Collection[FederationProtocol] {
	return resource.NewCollection(resource.Adapter[FederationProtocol]{
		Kind: "federationprotocols",
		Get: func(ctx context.Context, id string) (*FederationProtocol, error) {
			return a.Get(ctx, identityProviderID, id)
		},
		ID: func(v *FederationProtocol) string { return v.ID },
	})
}
