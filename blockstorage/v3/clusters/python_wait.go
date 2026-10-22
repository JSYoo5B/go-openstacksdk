package clusters

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET clusters/{name} for the
// shared wait engine, such as blockstorage.WaitForState and
// blockstorage.WaitForDelete. The reference is the cluster name; only
// resource.ID references are supported. Status reads the cluster status
// attribute, so a "deleted" status also completes a deletion wait.
func (a *API) Resources() *resource.Collection[Cluster] {
	return resource.NewCollection(resource.Adapter[Cluster]{
		Kind: "clusters",
		Get:  func(ctx context.Context, name string) (*Cluster, error) { return a.Get(ctx, name) },
		ID:   func(v *Cluster) string { return v.Name },
		Status: func(v *Cluster) string {
			if v.Status == nil {
				return ""
			}
			return *v.Status
		},
	})
}
