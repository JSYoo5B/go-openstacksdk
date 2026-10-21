package users

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/pybodyfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/db/v1/users"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// FilteredResources lists every account of this instance, like the pinned
// Python users(), and additionally accepts resource.WithBodyFilter and
// resource.WithBodyFilters for the Body attributes of the Python User: id,
// name, databases and password. Filters run locally on the original response
// rows with Python's rules (missing field is null, an object filter matches a
// subset, arrays compare exactly). Unknown names fail with
// resource.ErrInvalidOption before HTTP and server query extensions are not
// supported. The scope's WithHost narrowing is not applied here because Python
// has no host attribute; Get and deletion keep their host-aware behavior.
func (s *UserScope) FilteredResources() *resource.Collection[UserResource] {
	return pybodyfilter.Collection(pybodyfilter.Spec[UserResource]{
		Kind: "users", Envelope: "users",
		Pager:   func() pagination.Pager { return upstream.List(s.api.client, s.instanceID) },
		Extract: extractScopedUsers,
		ID:      func(v *UserResource) string { return v.Name }, Name: func(v *UserResource) string { return v.Name },
		Aliases: pybodyfilter.Aliases([]string{"id", "name", "databases", "password"}, nil),
	})
}
