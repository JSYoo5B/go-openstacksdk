package defaulttypes

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Resources returns a Collection bound to direct GET default-types/{project_id}
// for the shared wait engine, such as blockstorage.WaitForDelete. The reference
// is the project ID; only resource.ID references are supported. DefaultType has
// no status attribute; status waits require resource.WithStatusAttribute.
func (a *API) Resources() *resource.Collection[DefaultType] {
	return resource.NewCollection(resource.Adapter[DefaultType]{
		Kind: "default_types",
		Get:  func(ctx context.Context, projectID string) (*DefaultType, error) { return a.Get(ctx, projectID) },
		ID: func(v *DefaultType) string {
			if v.ProjectID == nil {
				return ""
			}
			return *v.ProjectID
		},
	})
}
