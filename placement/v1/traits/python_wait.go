package traits

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Trait is one custom or standard trait name.
type Trait struct{ Name string }

// Resources returns a Collection bound to direct GET and DELETE by trait name,
// so WaitDeleted completes on 404 as Python wait_for_delete does for Trait.
// Listing and name lookup are not bound; use API.List for enumeration. Traits
// have no status field, so Wait is unsupported.
func (a *API) Resources() *resource.Collection[Trait] {
	return resource.NewCollection(resource.Adapter[Trait]{
		Kind: "traits",
		Get: func(ctx context.Context, name string) (*Trait, error) {
			if err := a.Get(ctx, name); err != nil {
				return nil, err
			}
			return &Trait{Name: name}, nil
		},
		Delete: func(ctx context.Context, name string) error { return a.Delete(ctx, name) },
		ID:     func(v *Trait) string { return v.Name },
		Name:   func(v *Trait) string { return v.Name },
	})
}
