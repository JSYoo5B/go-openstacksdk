package resourceproviders

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// InventoryItem is one resource class inventory of a provider.
type InventoryItem struct {
	ResourceClass string
	*ResourceProviderInventory
}

// InventoryResources returns a Collection over the inventories of one fixed
// resource provider. Only direct GET by resource class is bound, so Find with
// resource.ID, WaitDeleted (completes on 404, as Python wait_for_delete does for
// ResourceProviderInventory) and Delete are available; listing and name lookup
// are not. Inventories have no string status field, so Wait with a status
// attribute is unsupported.
func (a *API) InventoryResources(resourceProviderID string) *resource.Collection[InventoryItem] {
	return resource.NewCollection(resource.Adapter[InventoryItem]{
		Kind: "inventories",
		Get: func(ctx context.Context, class string) (*InventoryItem, error) {
			value, err := a.GetInventory(ctx, resourceProviderID, class)
			if err != nil {
				return nil, err
			}
			return &InventoryItem{ResourceClass: class, ResourceProviderInventory: value}, nil
		},
		Delete: func(ctx context.Context, class string) error {
			return a.DeleteInventory(ctx, resourceProviderID, class)
		},
		ID:   func(v *InventoryItem) string { return v.ResourceClass },
		Name: func(v *InventoryItem) string { return v.ResourceClass },
	})
}
