package localips

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET local_ips/{id} for the
// shared wait engine, such as network.WaitForStatus and network.WaitForDelete.
// Only resource.ID references are supported; listing, name lookup and
// deletion stay on API. LocalIP has no status attribute, so status waits
// require resource.WithStatusAttribute with an exported string field.
func (a *API) Resources() *resource.Collection[LocalIP] {
	return resource.NewCollection(resource.Adapter[LocalIP]{
		Kind: "localips",
		Get:  func(ctx context.Context, id string) (*LocalIP, error) { return a.Get(ctx, id) },
		ID:   func(v *LocalIP) string { return v.ID },
	})
}
