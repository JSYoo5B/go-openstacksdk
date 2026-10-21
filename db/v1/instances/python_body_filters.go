package instances

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/pybodyfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/db/v1/instances"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// FilteredResources lists instances like Resources and additionally accepts
// resource.WithBodyFilter/WithBodyFilters for the Body attributes of the
// pinned Python Instance: id, name, status, flavor, volume, datastore, links,
// region, hostname, created_at (wire "created") and updated_at (wire
// "updated"). Wire names created and updated are accepted as aliases. Filters
// run locally on the original response rows with Python's rules: a missing
// field is null, an object filter matches a subset of a nonempty object, and
// other values compare exactly. Unknown names fail with resource.ErrInvalidOption
// before HTTP; server query extensions are not supported on this collection.
func (a *API) FilteredResources() *resource.Collection[Instance] {
	return pybodyfilter.Collection(pybodyfilter.Spec[Instance]{
		Kind: "instances", Envelope: "instances",
		Pager: func() pagination.Pager { return upstream.List(a.client) },
		Extract: func(page pagination.Page) ([]Instance, error) {
			values, err := upstream.ExtractInstances(page)
			return []Instance(values), err
		},
		ID: func(v *Instance) string { return v.ID }, Name: func(v *Instance) string { return v.Name },
		Get: func(ctx context.Context, id string) (*Instance, error) { return a.Get(ctx, id) },
		Aliases: pybodyfilter.Aliases(
			[]string{"id", "name", "status", "flavor", "volume", "datastore", "links", "region", "hostname"},
			map[string]string{"created_at": "created", "updated_at": "updated"}),
	})
}
