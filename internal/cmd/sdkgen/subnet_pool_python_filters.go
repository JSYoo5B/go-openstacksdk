package main

import "reflect"

const subnetPoolPythonResource = "openstack.network.v2.subnet_pool.SubnetPool"
const subnetPoolFilterManifestPath = "api/openstacksdk/resources/network/v2/subnet_pool.json"

var subnetPoolFilterSourceHashes = map[string]string{
	"openstack/network/v2/subnet_pool.py": "4028662600878ae10d80f64d97faafa48e3c0b4e603be48530f6488517a74fd0",
	"openstack/resource.py":               "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0",
	"openstack/fields.py":                 "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd",
	"openstack/proxy.py":                  "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46",
	"openstack/network/v2/_proxy.py":      "e409ee081f8b9c44903a1b81a4d9f9590c59f9f947aa686f960a5171ab1a6b58",
	"openstack/common/tag.py":             "c86f1d41fc1ed139366f163e4d5dfb7a37b15380dda05c3382c47dd7bafaa59e",
	"openstack/network/v2/_base.py":       "70981b01c16f24f656291068f808e97a5f9f309cfcda535a03056eccb54acc00",
}

func subnetPoolPythonQueryFields() map[string]string {
	return map[string]string{"address_scope_id": "address_scope_id", "any_tags": "tags-any", "description": "description", "fields": "fields", "ip_version": "ip_version", "is_default": "is_default", "is_shared": "shared", "limit": "limit", "marker": "marker", "name": "name", "not_any_tags": "not-tags-any", "not_tags": "not-tags", "project_id": "project_id", "sort_dir": "sort_dir", "sort_key": "sort_key", "tags": "tags"}
}

func subnetPoolPythonBodyFields() map[string]pythonFilterField {
	integer, list := "int", "list"
	return map[string]pythonFilterField{"created_at": {Field: "created_at"}, "default_prefix_length": {Field: "default_prefixlen", ResponseType: &integer}, "default_quota": {Field: "default_quota", ResponseType: &integer}, "id": {Field: "id"}, "maximum_prefix_length": {Field: "max_prefixlen", ResponseType: &integer}, "minimum_prefix_length": {Field: "min_prefixlen", ResponseType: &integer}, "prefixes": {Field: "prefixes", ResponseType: &list}, "revision_number": {Field: "revision_number", ResponseType: &integer}, "tenant_id": {Field: "tenant_id"}, "updated_at": {Field: "updated_at"}}
}

func subnetPoolPythonFilterMetadataValid(manifest *pythonFilterManifest) bool {
	if manifest == nil || manifest.SchemaVersion != 1 || manifest.SourcePin != pythonFilterPin || manifest.Resource != subnetPoolPythonResource || manifest.SDKPackage != "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools" || manifest.BasePath != "/subnetpools" || manifest.Envelope != "subnetpools" || manifest.Counts.CanonicalQuery != 16 || manifest.Counts.AcceptedQuery != 20 || manifest.Counts.LocalBody != 10 || !reflect.DeepEqual(manifest.Query, subnetPoolPythonQueryFields()) || !reflect.DeepEqual(manifest.Body, subnetPoolPythonBodyFields()) || len(manifest.QueryFormats) != 0 || len(manifest.URI) != 0 || manifest.UnknownFilters != "discard" || manifest.QueryCollision != "canonical_client_name_wins" || !reflect.DeepEqual(manifest.ClassBases, []string{"resource.Resource", "_base.TagMixinNetwork"}) || !reflect.DeepEqual(manifest.MRO, []string{subnetPoolPythonResource, "openstack.resource.Resource", "builtins.dict", "openstack.network.v2._base.TagMixinNetwork", "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol", "typing.Protocol"}) || len(manifest.Proof.Nodes) != 33 {
		return false
	}
	wanted := map[string]string{"SubnetPool.project_id": "openstack/network/v2/subnet_pool.py", "SubnetPool.tenant_id": "openstack/network/v2/subnet_pool.py", "SubnetPool.prefixes": "openstack/network/v2/subnet_pool.py", "SubnetPool.default_prefix_length": "openstack/network/v2/subnet_pool.py", "SubnetPool.default_quota": "openstack/network/v2/subnet_pool.py", "SubnetPool.maximum_prefix_length": "openstack/network/v2/subnet_pool.py", "SubnetPool.minimum_prefix_length": "openstack/network/v2/subnet_pool.py", "SubnetPool.revision_number": "openstack/network/v2/subnet_pool.py", "SubnetPool.created_at": "openstack/network/v2/subnet_pool.py", "SubnetPool.updated_at": "openstack/network/v2/subnet_pool.py", "TagMixinNetwork": "openstack/network/v2/_base.py", "TagMixin._tag_query_parameters": "openstack/common/tag.py", "TagMixin.tags": "openstack/common/tag.py", "Resource.id": "openstack/resource.py", "Resource.__getattribute__": "openstack/resource.py", "QueryParameters.__init__": "openstack/resource.py", "_BaseComponent.__init__": "openstack/fields.py", "_convert_type": "openstack/fields.py", "Resource.list._dict_filter": "openstack/resource.py"}
	for _, node := range manifest.Proof.Nodes {
		if path, ok := wanted[node.Symbol]; ok && path == node.Source {
			delete(wanted, node.Symbol)
		}
	}
	return len(wanted) == 0
}

func verifySubnetPoolPythonFilterManifest(source string, manifest *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, manifest, "SubnetPool", subnetPoolPythonResource, subnetPoolFilterSourceHashes, subnetPoolPythonFilterMetadataValid)
}

func loadSubnetPoolPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "SubnetPool", subnetPoolFilterManifestPath, verifySubnetPoolPythonFilterManifest)
}
