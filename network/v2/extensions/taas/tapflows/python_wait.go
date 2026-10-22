package tapflows

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET taas/tap_flows/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. TapFlow declares Status, so network.WaitForStatus selects it by default.
func (a *API) Resources() *resource.Collection[TapFlow] {
	return resource.NewCollection(resource.Adapter[TapFlow]{
		Kind: "tapflows",
		Get:  func(ctx context.Context, id string) (*TapFlow, error) { return a.Get(ctx, id) },
		ID:   func(v *TapFlow) string { return v.ID },
	})
}
