package resourceproviders

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ProviderTraits is the trait set of one resource provider, the Go form of
// Python ResourceProviderTrait. ResourceProviderID is the polled reference.
type ProviderTraits struct {
	ResourceProviderID string
	*ResourceProviderTraits
}

// TraitResources returns a Collection bound to direct GET
// resource_providers/{id}/traits, keyed by resource provider ID, for the
// shared wait engine, such as placement.WaitForDelete. The route answers 404
// only when the provider itself is missing. Only resource.ID references are
// supported; the trait set has no string attribute, so status waits are
// unsupported. Use API.DeleteTraits to dissociate traits.
func (a *API) TraitResources() *resource.Collection[ProviderTraits] {
	return resource.NewCollection(resource.Adapter[ProviderTraits]{
		Kind: "resourceprovidertraits",
		Get: func(ctx context.Context, id string) (*ProviderTraits, error) {
			value, err := a.GetTraits(ctx, id)
			if err != nil || value == nil {
				return nil, err
			}
			return &ProviderTraits{ResourceProviderID: id, ResourceProviderTraits: value}, nil
		},
		ID: func(v *ProviderTraits) string { return v.ResourceProviderID },
	})
}
