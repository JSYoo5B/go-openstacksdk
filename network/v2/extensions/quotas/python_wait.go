package quotas

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET quotas/{project_id} for
// the shared wait engine, such as network.WaitForDelete. The reference is the
// project ID; only resource.ID references are supported. Quota has only
// numeric attributes, so status waits are unsupported. Neutron reports
// defaults for a project without overrides, so a deletion wait completes only
// on an actual 404.
func (a *API) Resources() *resource.Collection[Quota] {
	return resource.NewCollection(resource.Adapter[Quota]{
		Kind: "quotas",
		Get:  func(ctx context.Context, projectID string) (*Quota, error) { return a.Get(ctx, projectID) },
		// Quota carries no project ID. ID is used only for Name references,
		// which this collection does not support.
		ID: func(*Quota) string { return "" },
	})
}
