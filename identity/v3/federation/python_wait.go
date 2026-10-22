package federation

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// MappingResources returns a Collection bound to direct GET
// OS-FEDERATION/mappings/{id} for the shared wait engine, such as
// identity.WaitForDelete. Only resource.ID references are supported. Mapping
// has no status attribute.
func (a *API) MappingResources() *resource.Collection[Mapping] {
	return resource.NewCollection(resource.Adapter[Mapping]{
		Kind: "mappings",
		Get:  func(ctx context.Context, id string) (*Mapping, error) { return a.GetMapping(ctx, id) },
		ID:   func(v *Mapping) string { return v.ID },
	})
}
