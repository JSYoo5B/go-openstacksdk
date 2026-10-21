package databases

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/pybodyfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/db/v1/databases"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// FilteredResources lists this instance's databases like the scope and
// additionally accepts resource.WithBodyFilter/WithBodyFilters for the Body
// attributes of the pinned Python Database: id, name, character_set and
// collate. Filters run locally on the original response rows with Python's
// rules (missing field is null; other values compare exactly). Unknown names
// fail with resource.ErrInvalidOption before HTTP, and server query
// extensions are not supported. Get still searches the instance list.
func (s *DatabaseScope) FilteredResources() *resource.Collection[Database] {
	return pybodyfilter.Collection(pybodyfilter.Spec[Database]{
		Kind: "databases", Envelope: "databases",
		Pager:   func() pagination.Pager { return upstream.List(s.api.client, s.instanceID) },
		Extract: extractScopedDatabases,
		ID:      func(v *Database) string { return v.Name }, Name: func(v *Database) string { return v.Name },
		Aliases: pybodyfilter.Aliases([]string{"id", "name", "character_set", "collate"}, nil),
	})
}
