package extensions

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	common "github.com/gophercloud/gophercloud/v2/openstack/common/extensions"
)

// Resources returns a Collection of the common extension model bound to
// direct GET extensions/{alias} for
// the shared wait engine, such as network.WaitForDelete. The reference is the
// extension alias; only resource.ID references are supported. Extension has no
// status attribute; status waits require resource.WithStatusAttribute.
func (a *API) Resources() *resource.Collection[common.Extension] {
	return resource.NewCollection(resource.Adapter[common.Extension]{
		Kind: "extensions",
		Get:  func(ctx context.Context, alias string) (*common.Extension, error) { return a.Get(ctx, alias) },
		ID:   func(v *common.Extension) string { return v.Alias },
	})
}
