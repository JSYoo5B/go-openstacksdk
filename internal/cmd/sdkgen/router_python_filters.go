package main

import "reflect"

const routerSDKPath = "network/v2/extensions/layer3/routers"
const routerPythonResource = "openstack.network.v2.router.Router"
const routerFilterManifestPath = "api/openstacksdk/resources/network/v2/router.json"

var routerFilterSourceHashes = map[string]string{"openstack/common/tag.py": "c86f1d41fc1ed139366f163e4d5dfb7a37b15380dda05c3382c47dd7bafaa59e", "openstack/fields.py": "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd", "openstack/network/v2/_base.py": "70981b01c16f24f656291068f808e97a5f9f309cfcda535a03056eccb54acc00", "openstack/network/v2/_proxy.py": "e409ee081f8b9c44903a1b81a4d9f9590c59f9f947aa686f960a5171ab1a6b58", "openstack/network/v2/router.py": "fdb6589856a606bbd173f52eb1d549e2028eca9dd6bd9dddd3c76d8efe3ae1f8", "openstack/proxy.py": "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46", "openstack/resource.py": "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0"}

func routerPythonQueryFields() map[string]string {
	return map[string]string{"any_tags": "tags-any", "description": "description", "fields": "fields", "flavor_id": "flavor_id", "id": "id", "is_admin_state_up": "admin_state_up", "is_distributed": "distributed", "is_ha": "ha", "limit": "limit", "marker": "marker", "name": "name", "not_any_tags": "not-tags-any", "not_tags": "not-tags", "project_id": "project_id", "sort_dir": "sort_dir", "sort_key": "sort_key", "status": "status", "tags": "tags"}
}
func routerPythonBodyFields() map[string]pythonFilterField {
	bool, list, int, dict := "bool", "list", "int", "dict"
	return map[string]pythonFilterField{"availability_zone_hints": {Field: "availability_zone_hints", ResponseType: &list}, "availability_zones": {Field: "availability_zones", ResponseType: &list}, "created_at": {Field: "created_at"}, "enable_ndp_proxy": {Field: "enable_ndp_proxy", ResponseType: &bool}, "evpn_vni": {Field: "evpn_vni", ResponseType: &int}, "external_gateway_info": {Field: "external_gateway_info", ResponseType: &dict}, "revision_number": {Field: "revision", ResponseType: &int}, "routes": {Field: "routes", ResponseType: &list}, "tenant_id": {Field: "tenant_id"}, "updated_at": {Field: "updated_at"}}
}
func routerPythonFilterMetadataValid(m *pythonFilterManifest) bool {
	if m == nil || m.SchemaVersion != 1 || m.SourcePin != pythonFilterPin || m.Resource != routerPythonResource || m.SDKPackage != "gophercloudsdk/network/v2/extensions/layer3/routers" || m.BasePath != "/routers" || m.Envelope != "routers" || m.Counts.CanonicalQuery != 18 || m.Counts.AcceptedQuery != 24 || m.Counts.LocalBody != 10 || !reflect.DeepEqual(m.Query, routerPythonQueryFields()) || !reflect.DeepEqual(m.Body, routerPythonBodyFields()) || len(m.QueryFormats) != 0 || len(m.URI) != 0 || m.UnknownFilters != "discard" || m.QueryCollision != "canonical_client_name_wins" || !reflect.DeepEqual(m.ClassBases, []string{"_base.NetworkResource", "_base.TagMixinNetwork"}) || !reflect.DeepEqual(m.MRO, []string{"openstack.network.v2.router.Router", "openstack.network.v2._base.NetworkResource", "openstack.resource.Resource", "builtins.dict", "openstack.network.v2._base.TagMixinNetwork", "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol", "typing.Protocol"}) || len(m.Proof.Nodes) != 41 {
		return false
	}
	wanted := map[string]string{"NetworkResource": "openstack/network/v2/_base.py", "NetworkResource.revision_number": "openstack/network/v2/_base.py", "Proxy._list": "openstack/proxy.py", "Proxy.routers": "openstack/network/v2/_proxy.py", "QueryParameters.__init__": "openstack/resource.py", "QueryParameters._transpose": "openstack/resource.py", "QueryParameters._validate": "openstack/resource.py", "Resource": "openstack/resource.py", "Resource.__getattribute__": "openstack/resource.py", "Resource.__getitem__": "openstack/resource.py", "Resource.__init__": "openstack/resource.py", "Resource._alternate_id": "openstack/resource.py", "Resource._attributes_iterator": "openstack/resource.py", "Resource._collect_attrs": "openstack/resource.py", "Resource._get_next_link": "openstack/resource.py", "Resource._max_microversion": "openstack/resource.py", "Resource.id": "openstack/resource.py", "Resource.list": "openstack/resource.py", "Resource.name": "openstack/resource.py", "Resource.to_dict": "openstack/resource.py", "ResourceMixinProtocol": "openstack/resource.py", "Router": "openstack/network/v2/router.py", "Router._query_mapping": "openstack/network/v2/router.py", "Router.availability_zone_hints": "openstack/network/v2/router.py", "Router.availability_zones": "openstack/network/v2/router.py", "Router.created_at": "openstack/network/v2/router.py", "Router.enable_ndp_proxy": "openstack/network/v2/router.py", "Router.evpn_vni": "openstack/network/v2/router.py", "Router.external_gateway_info": "openstack/network/v2/router.py", "Router.project_id": "openstack/network/v2/router.py", "Router.revision_number": "openstack/network/v2/router.py", "Router.routes": "openstack/network/v2/router.py", "Router.tenant_id": "openstack/network/v2/router.py", "Router.updated_at": "openstack/network/v2/router.py", "TagMixin": "openstack/common/tag.py", "TagMixin._tag_query_parameters": "openstack/common/tag.py", "TagMixin.tags": "openstack/common/tag.py", "TagMixinNetwork": "openstack/network/v2/_base.py", "_BaseComponent.__get__": "openstack/fields.py", "_BaseComponent.__init__": "openstack/fields.py", "_convert_type": "openstack/fields.py"}
	for _, node := range m.Proof.Nodes {
		if path, ok := wanted[node.Symbol]; ok && path == node.Source {
			delete(wanted, node.Symbol)
		}
	}
	return len(wanted) == 0
}
func verifyRouterPythonFilterManifest(source string, m *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, m, "Router", routerPythonResource, routerFilterSourceHashes, routerPythonFilterMetadataValid)
}
func loadRouterPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "Router", routerFilterManifestPath, verifyRouterPythonFilterManifest)
}
