package clustertemplates

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/pybodyfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ListRecord is one row of PythonResources. ClusterTemplate is the native
// model and the embedded Metadata keeps the original row fields (Body),
// response headers and status.
type ListRecord struct {
	resource.Metadata
	ClusterTemplate ClusterTemplate
}

func (r *ListRecord) UnmarshalJSON(data []byte) error {
	var value ListRecord
	if err := resource.DecodeObject(data, &value.ClusterTemplate, &value.Metadata); err != nil {
		return err
	}
	*r = value
	return nil
}

var pythonBodyKinds = map[string]resource.BodyFieldType{
	"apiserver_port": resource.BodyFieldInteger, "docker_volume_size": resource.BodyFieldInteger,
	"hidden": resource.BodyFieldBoolean, "master_lb_enabled": resource.BodyFieldBoolean, "tls_disabled": resource.BodyFieldBoolean,
	"public": resource.BodyFieldBoolean, "registry_enabled": resource.BodyFieldBoolean,
}

// PythonResources lists GET /clustertemplates like the pinned Python
// cluster_templates() generator. Continuation follows body links, a truthy next
// field and the Link header; when an explicit page size (resource.WithPageSize
// or resource.WithQuery("limit", n)) was sent, a nonempty page without a link
// is followed by marker=<id or uuid of the last row>, and an empty page ends
// the list. resource.WithBodyFilter/WithBodyFilters apply Python's client-side
// Body-attribute filters to the original rows by the Python attribute names
// (is_public, is_hidden, is_tls_disabled, is_registry_enabled,
// is_master_lb_enabled, is_floating_ip_enabled and the wire names). Unknown
// attribute names fail with resource.ErrInvalidOption before HTTP. Python
// declares floating_ip_enabled without a boolean type, so it compares as raw JSON.
func (a *API) PythonResources() *resource.Collection[ListRecord] {
	spec := rest.CollectionSpec[ListRecord]{
		Client: a.client, Path: "clustertemplates", Kind: "clustertemplates", SingleKey: "clustertemplate", PluralKey: "clustertemplates",
		ID:       func(r *ListRecord) string { return r.ClusterTemplate.UUID },
		Name:     func(r *ListRecord) string { return r.ClusterTemplate.Name },
		Metadata: func(r *ListRecord) *resource.Metadata { return &r.Metadata },
		Paging:   pybodyfilter.PythonPaging(func(r *ListRecord) *resource.Metadata { return &r.Metadata }, "id", "uuid"),
	}
	return pybodyfilter.REST(spec, pybodyfilter.Aliases(
		[]string{"id", "name", "apiserver_port", "cluster_distro", "coe", "created_at", "docker_storage_driver", "docker_volume_size",
			"dns_nameserver", "external_network_id", "fixed_network", "fixed_subnet", "flavor_id", "floating_ip_enabled", "http_proxy",
			"https_proxy", "image_id", "insecure_registry", "keypair_id", "labels", "master_flavor_id", "network_driver", "no_proxy",
			"server_type", "updated_at", "uuid", "volume_driver", "hidden", "master_lb_enabled", "tls_disabled", "public", "registry_enabled"},
		map[string]string{"is_floating_ip_enabled": "floating_ip_enabled", "is_master_lb_enabled": "master_lb_enabled", "is_hidden": "hidden",
			"is_tls_disabled": "tls_disabled", "is_public": "public", "is_registry_enabled": "registry_enabled"}),
		pythonBodyKinds)
}
