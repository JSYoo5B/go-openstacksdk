package labels

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET metering/metering-labels/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. MeteringLabel has no status attribute; status waits require resource.WithStatusAttribute with an exported string field.
func (a *API) Resources() *resource.Collection[MeteringLabel] {
	return resource.NewCollection(resource.Adapter[MeteringLabel]{
		Kind: "meteringlabels",
		Get:  func(ctx context.Context, id string) (*MeteringLabel, error) { return a.Get(ctx, id) },
		ID:   func(v *MeteringLabel) string { return v.ID },
	})
}
