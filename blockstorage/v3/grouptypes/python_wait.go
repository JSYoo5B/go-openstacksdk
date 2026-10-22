package grouptypes

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET group_types/{id} for the
// shared wait engine, such as blockstorage.WaitForDelete. Only resource.ID
// references are supported. GroupType has no status attribute; status waits
// require resource.WithStatusAttribute.
func (a *API) Resources() *resource.Collection[GroupType] {
	return resource.NewCollection(resource.Adapter[GroupType]{
		Kind: "group_types",
		Get:  func(ctx context.Context, id string) (*GroupType, error) { return a.Get(ctx, id) },
		ID:   func(v *GroupType) string { return v.ID },
	})
}
