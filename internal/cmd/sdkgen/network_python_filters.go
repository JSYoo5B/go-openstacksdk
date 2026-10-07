package main

import "reflect"

const networkSDKPath = "network/v2/networks"
const networkPythonResource = "openstack.network.v2.network.Network"
const networkFilterManifestPath = "api/openstacksdk/resources/network/v2/network.json"

var networkFilterSourceHashes = map[string]string{"openstack/common/tag.py": "c86f1d41fc1ed139366f163e4d5dfb7a37b15380dda05c3382c47dd7bafaa59e", "openstack/fields.py": "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd", "openstack/network/v2/_base.py": "70981b01c16f24f656291068f808e97a5f9f309cfcda535a03056eccb54acc00", "openstack/network/v2/_proxy.py": "e409ee081f8b9c44903a1b81a4d9f9590c59f9f947aa686f960a5171ab1a6b58", "openstack/network/v2/network.py": "ad17c5dadc94a28f35d6f0c95fa95b2a15165b12ab652c1be188ba30aeca7500", "openstack/proxy.py": "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46", "openstack/resource.py": "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0"}

func networkPythonQueryFields() map[string]string {
	return map[string]string{"any_tags": "tags-any", "description": "description", "fields": "fields", "id": "id", "ipv4_address_scope_id": "ipv4_address_scope", "ipv6_address_scope_id": "ipv6_address_scope", "is_admin_state_up": "admin_state_up", "is_port_security_enabled": "port_security_enabled", "is_router_external": "router:external", "is_shared": "shared", "limit": "limit", "marker": "marker", "name": "name", "not_any_tags": "not-tags-any", "not_tags": "not-tags", "project_id": "project_id", "provider_network_type": "provider:network_type", "provider_physical_network": "provider:physical_network", "provider_segmentation_id": "provider:segmentation_id", "sort_dir": "sort_dir", "sort_key": "sort_key", "status": "status", "tags": "tags"}
}
func networkPythonBodyFields() map[string]pythonFilterField {
	bool, list, int := "bool", "list", "int"
	return map[string]pythonFilterField{"availability_zone_hints": {Field: "availability_zone_hints", ResponseType: &list}, "availability_zones": {Field: "availability_zones", ResponseType: &list}, "created_at": {Field: "created_at"}, "dns_domain": {Field: "dns_domain"}, "is_default": {Field: "is_default", ResponseType: &bool}, "is_vlan_qinq": {Field: "vlan_qinq", ResponseType: &bool}, "is_vlan_transparent": {Field: "vlan_transparent", ResponseType: &bool}, "mtu": {Field: "mtu", ResponseType: &int}, "pvlan": {Field: "pvlan", ResponseType: &bool}, "qos_policy_id": {Field: "qos_policy_id"}, "revision_number": {Field: "revision_number", ResponseType: &int}, "segments": {Field: "segments", ResponseType: &list}, "subnet_ids": {Field: "subnets", ResponseType: &list}, "updated_at": {Field: "updated_at"}}
}
func networkPythonFilterMetadataValid(m *pythonFilterManifest) bool {
	if m == nil || m.SchemaVersion != 1 || m.SourcePin != pythonFilterPin || m.Resource != networkPythonResource || m.SDKPackage != "github.com/JSYoo5B/gophercloudsdk/network/v2/networks" || m.BasePath != "/networks" || m.Envelope != "networks" || m.Counts.CanonicalQuery != 23 || m.Counts.AcceptedQuery != 35 || m.Counts.LocalBody != 14 || !reflect.DeepEqual(m.Query, networkPythonQueryFields()) || !reflect.DeepEqual(m.Body, networkPythonBodyFields()) || len(m.QueryFormats) != 0 || len(m.URI) != 0 || m.UnknownFilters != "discard" || m.QueryCollision != "canonical_client_name_wins" || !reflect.DeepEqual(m.ClassBases, []string{"_base.NetworkResource", "_base.TagMixinNetwork"}) || !reflect.DeepEqual(m.MRO, []string{"openstack.network.v2.network.Network", "openstack.network.v2._base.NetworkResource", "openstack.resource.Resource", "builtins.dict", "openstack.network.v2._base.TagMixinNetwork", "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol", "typing.Protocol"}) || len(m.Proof.Nodes) != 43 {
		return false
	}
	wanted := map[string]string{"Network": "openstack/network/v2/network.py", "Network._query_mapping": "openstack/network/v2/network.py", "Network.availability_zone_hints": "openstack/network/v2/network.py", "Network.availability_zones": "openstack/network/v2/network.py", "Network.created_at": "openstack/network/v2/network.py", "Network.dns_domain": "openstack/network/v2/network.py", "Network.is_default": "openstack/network/v2/network.py", "Network.is_vlan_qinq": "openstack/network/v2/network.py", "Network.is_vlan_transparent": "openstack/network/v2/network.py", "Network.mtu": "openstack/network/v2/network.py", "Network.pvlan": "openstack/network/v2/network.py", "Network.qos_policy_id": "openstack/network/v2/network.py", "Network.segments": "openstack/network/v2/network.py", "Network.subnet_ids": "openstack/network/v2/network.py", "Network.updated_at": "openstack/network/v2/network.py", "NetworkResource": "openstack/network/v2/_base.py", "NetworkResource.revision_number": "openstack/network/v2/_base.py", "Proxy._list": "openstack/proxy.py", "Proxy.networks": "openstack/network/v2/_proxy.py", "QueryParameters.__init__": "openstack/resource.py", "QueryParameters._transpose": "openstack/resource.py", "QueryParameters._validate": "openstack/resource.py", "Resource": "openstack/resource.py", "Resource.__getattribute__": "openstack/resource.py", "Resource.__getitem__": "openstack/resource.py", "Resource.__init__": "openstack/resource.py", "Resource._alternate_id": "openstack/resource.py", "Resource._attributes_iterator": "openstack/resource.py", "Resource._collect_attrs": "openstack/resource.py", "Resource._get_next_link": "openstack/resource.py", "Resource._max_microversion": "openstack/resource.py", "Resource.id": "openstack/resource.py", "Resource.list": "openstack/resource.py", "Resource.name": "openstack/resource.py", "Resource.to_dict": "openstack/resource.py", "ResourceMixinProtocol": "openstack/resource.py", "TagMixin": "openstack/common/tag.py", "TagMixin._tag_query_parameters": "openstack/common/tag.py", "TagMixin.tags": "openstack/common/tag.py", "TagMixinNetwork": "openstack/network/v2/_base.py", "_BaseComponent.__get__": "openstack/fields.py", "_BaseComponent.__init__": "openstack/fields.py", "_convert_type": "openstack/fields.py"}
	for _, node := range m.Proof.Nodes {
		if path, ok := wanted[node.Symbol]; ok && path == node.Source {
			delete(wanted, node.Symbol)
		}
	}
	return len(wanted) == 0
}
func verifyNetworkPythonFilterManifest(source string, m *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, m, "Network", networkPythonResource, networkFilterSourceHashes, networkPythonFilterMetadataValid)
}
func loadNetworkPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "Network", networkFilterManifestPath, verifyNetworkPythonFilterManifest)
}
