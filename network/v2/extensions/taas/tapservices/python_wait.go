package tapservices

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET taas/tap_services/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. TapService declares Status, so network.WaitForStatus selects it by default.
func (a *API) Resources() *resource.Collection[TapService] {
	return resource.NewCollection(resource.Adapter[TapService]{
		Kind: "tapservices",
		Get:  func(ctx context.Context, id string) (*TapService, error) { return a.Get(ctx, id) },
		ID:   func(v *TapService) string { return v.ID },
	})
}
