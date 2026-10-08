package main

import "reflect"

const trunkSDKPath = "network/v2/extensions/trunks"
const trunkPythonResource = "openstack.network.v2.trunk.Trunk"
const trunkFilterManifestPath = "api/openstacksdk/resources/network/v2/trunk.json"

var trunkFilterSourceHashes = map[string]string{"openstack/common/tag.py": "c86f1d41fc1ed139366f163e4d5dfb7a37b15380dda05c3382c47dd7bafaa59e", "openstack/fields.py": "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd", "openstack/network/v2/_proxy.py": "e409ee081f8b9c44903a1b81a4d9f9590c59f9f947aa686f960a5171ab1a6b58", "openstack/network/v2/trunk.py": "0cfb0b32805ae8b5f70f97e06ac64240af69296c15b8c06a82741badad2d2028", "openstack/proxy.py": "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46", "openstack/resource.py": "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0"}

func trunkPythonQueryFields() map[string]string {
	return map[string]string{"any_tags": "tags-any", "description": "description", "fields": "fields", "is_admin_state_up": "admin_state_up", "limit": "limit", "marker": "marker", "name": "name", "not_any_tags": "not-tags-any", "not_tags": "not-tags", "port_id": "port_id", "project_id": "project_id", "status": "status", "sub_ports": "sub_ports", "tags": "tags"}
}
func trunkPythonBodyFields() map[string]pythonFilterField {
	return map[string]pythonFilterField{"id": {Field: "id"}, "tenant_id": {Field: "tenant_id"}}
}

func trunkPythonFilterMetadataValid(m *pythonFilterManifest) bool {
	if m == nil || m.SchemaVersion != 1 || m.SourcePin != pythonFilterPin || m.Resource != trunkPythonResource || m.SDKPackage != "github.com/JSYoo5B/go-openstacksdk/"+trunkSDKPath || m.BasePath != "/trunks" || m.Envelope != "trunks" || m.Counts.CanonicalQuery != 14 || m.Counts.AcceptedQuery != 18 || m.Counts.LocalBody != 2 || !reflect.DeepEqual(m.Query, trunkPythonQueryFields()) || !reflect.DeepEqual(m.Body, trunkPythonBodyFields()) || len(m.QueryFormats) != 0 || len(m.URI) != 0 || m.UnknownFilters != "discard" || m.QueryCollision != "canonical_client_name_wins" || !reflect.DeepEqual(m.ClassBases, []string{"resource.Resource", "tag.TagMixin"}) || !reflect.DeepEqual(m.MRO, []string{"openstack.network.v2.trunk.Trunk", "openstack.resource.Resource", "builtins.dict", "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol", "typing.Protocol"}) || len(m.Proof.Nodes) != 35 {
		return false
	}
	wanted := map[string]string{"Proxy._list": "openstack/proxy.py", "Proxy.trunks": "openstack/network/v2/_proxy.py", "QueryParameters.__init__": "openstack/resource.py", "QueryParameters._transpose": "openstack/resource.py", "QueryParameters._validate": "openstack/resource.py", "Resource": "openstack/resource.py", "Resource.__getattribute__": "openstack/resource.py", "Resource.__getitem__": "openstack/resource.py", "Resource.__init__": "openstack/resource.py", "Resource._alternate_id": "openstack/resource.py", "Resource._attributes_iterator": "openstack/resource.py", "Resource._collect_attrs": "openstack/resource.py", "Resource._get_next_link": "openstack/resource.py", "Resource._max_microversion": "openstack/resource.py", "Resource.id": "openstack/resource.py", "Resource.list": "openstack/resource.py", "Resource.name": "openstack/resource.py", "Resource.to_dict": "openstack/resource.py", "ResourceMixinProtocol": "openstack/resource.py", "TagMixin": "openstack/common/tag.py", "TagMixin._tag_query_parameters": "openstack/common/tag.py", "TagMixin.tags": "openstack/common/tag.py", "Trunk": "openstack/network/v2/trunk.py", "Trunk._query_mapping": "openstack/network/v2/trunk.py", "Trunk.description": "openstack/network/v2/trunk.py", "Trunk.is_admin_state_up": "openstack/network/v2/trunk.py", "Trunk.name": "openstack/network/v2/trunk.py", "Trunk.port_id": "openstack/network/v2/trunk.py", "Trunk.project_id": "openstack/network/v2/trunk.py", "Trunk.status": "openstack/network/v2/trunk.py", "Trunk.sub_ports": "openstack/network/v2/trunk.py", "Trunk.tenant_id": "openstack/network/v2/trunk.py", "_BaseComponent.__get__": "openstack/fields.py", "_BaseComponent.__init__": "openstack/fields.py", "_convert_type": "openstack/fields.py"}
	for _, n := range m.Proof.Nodes {
		if p, ok := wanted[n.Symbol]; ok && p == n.Source {
			delete(wanted, n.Symbol)
		}
	}
	return len(wanted) == 0
}
func verifyTrunkPythonFilterManifest(source string, m *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, m, "Trunk", trunkPythonResource, trunkFilterSourceHashes, trunkPythonFilterMetadataValid)
}
func loadTrunkPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "Trunk", trunkFilterManifestPath, verifyTrunkPythonFilterManifest)
}
