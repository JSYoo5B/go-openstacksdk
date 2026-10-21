package clusters

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/pybodyfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ListRecord is one row of PythonResources. Cluster is the native model and
// the embedded Metadata keeps the original row fields (Body), response headers
// and status, so omitted and null summary fields stay distinguishable.
type ListRecord struct {
	resource.Metadata
	Cluster Cluster
}

func (r *ListRecord) UnmarshalJSON(data []byte) error {
	var value ListRecord
	if err := resource.DecodeObject(data, &value.Cluster, &value.Metadata); err != nil {
		return err
	}
	*r = value
	return nil
}

var pythonBodyKinds = map[string]resource.BodyFieldType{
	"create_timeout": resource.BodyFieldInteger, "master_count": resource.BodyFieldInteger, "node_count": resource.BodyFieldInteger,
	"floating_ip_enabled": resource.BodyFieldBoolean, "master_lb_enabled": resource.BodyFieldBoolean,
}

// PythonResources lists GET /clusters like the pinned Python clusters()
// generator. Continuation follows body links, a truthy next field and the Link
// header; when an explicit page size (resource.WithPageSize or
// resource.WithQuery("limit", n)) was sent, a nonempty page without a link is
// followed by marker=<id or uuid of the last row>, and an empty page ends the
// list. resource.WithBodyFilter/WithBodyFilters apply Python's client-side
// Body-attribute filters to the original rows by the Python attribute names
// (is_floating_ip_enabled, is_master_lb_enabled, status_reason and the wire
// names). Unknown attribute names fail with resource.ErrInvalidOption before
// HTTP. The route is the summary list, not /clusters/detail.
func (a *API) PythonResources() *resource.Collection[ListRecord] {
	spec := rest.CollectionSpec[ListRecord]{
		Client: a.client, Path: "clusters", Kind: "clusters", SingleKey: "cluster", PluralKey: "clusters",
		ID:       func(r *ListRecord) string { return r.Cluster.UUID },
		Name:     func(r *ListRecord) string { return r.Cluster.Name },
		Status:   func(r *ListRecord) string { return r.Cluster.Status },
		Metadata: func(r *ListRecord) *resource.Metadata { return &r.Metadata },
		Paging:   pybodyfilter.PythonPaging(func(r *ListRecord) *resource.Metadata { return &r.Metadata }, "id", "uuid"),
	}
	return pybodyfilter.REST(spec, pybodyfilter.Aliases(
		[]string{"id", "name", "api_address", "cluster_template_id", "coe_version", "create_timeout", "created_at", "discovery_url",
			"fixed_network", "fixed_subnet", "flavor_id", "keypair", "labels", "master_addresses", "master_count", "master_flavor_id",
			"node_count", "node_addresses", "stack_id", "status", "updated_at", "uuid", "project_id"},
		map[string]string{"is_floating_ip_enabled": "floating_ip_enabled", "is_master_lb_enabled": "master_lb_enabled", "status_reason": "reason"}),
		pythonBodyKinds)
}
