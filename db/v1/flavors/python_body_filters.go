package flavors

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/pybodyfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/db/v1/flavors"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// FilteredResources lists flavors like Resources and additionally accepts
// resource.WithBodyFilter/WithBodyFilters for the Body attributes the pinned
// Python Flavor declares: id, name, ram and links. Filters run locally on the
// original response rows, after the native pager, with Python's rules: a
// missing field is null, an object filter matches a subset of a nonempty
// object, other values compare exactly. Unknown attribute names fail with
// resource.ErrInvalidOption before HTTP; server query extensions are not
// supported on this collection.
func (a *API) FilteredResources() *resource.Collection[Flavor] {
	return pybodyfilter.Collection(pybodyfilter.Spec[Flavor]{
		Kind: "flavors", Envelope: "flavors",
		Pager: func() pagination.Pager { return upstream.List(a.client) },
		Extract: func(page pagination.Page) ([]Flavor, error) {
			values, err := upstream.ExtractFlavors(page)
			return []Flavor(values), err
		},
		ID: func(v *Flavor) string { return fmt.Sprint(v.ID) }, Name: func(v *Flavor) string { return v.Name },
		Get:     func(ctx context.Context, id string) (*Flavor, error) { return a.Get(ctx, id) },
		Aliases: pybodyfilter.Aliases([]string{"id", "name", "ram", "links"}, nil),
	})
}
