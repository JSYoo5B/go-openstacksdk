package rules

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET metering/metering-label-rules/{id} for the shared
// wait engine, such as network.WaitForStatus and network.WaitForDelete. Only
// resource.ID references are supported; listing, name lookup and deletion stay
// on API. MeteringLabelRule has no status attribute; status waits require resource.WithStatusAttribute with an exported string field such as Direction.
func (a *API) Resources() *resource.Collection[MeteringLabelRule] {
	return resource.NewCollection(resource.Adapter[MeteringLabelRule]{
		Kind: "meteringlabelrules",
		Get:  func(ctx context.Context, id string) (*MeteringLabelRule, error) { return a.Get(ctx, id) },
		ID:   func(v *MeteringLabelRule) string { return v.ID },
	})
}
