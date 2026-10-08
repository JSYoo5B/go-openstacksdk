package main

import "reflect"

const securityGroupSDKPath = "network/v2/extensions/security/groups"
const securityGroupRulesNativePath = upstreamModule + "/openstack/networking/v2/extensions/security/rules"
const securityGroupPythonResource = "openstack.network.v2.security_group.SecurityGroup"
const securityGroupFilterManifestPath = "api/openstacksdk/resources/network/v2/security_group.json"

var securityGroupFilterSourceHashes = map[string]string{"openstack/common/tag.py": "c86f1d41fc1ed139366f163e4d5dfb7a37b15380dda05c3382c47dd7bafaa59e", "openstack/fields.py": "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd", "openstack/network/v2/_base.py": "70981b01c16f24f656291068f808e97a5f9f309cfcda535a03056eccb54acc00", "openstack/network/v2/_proxy.py": "e409ee081f8b9c44903a1b81a4d9f9590c59f9f947aa686f960a5171ab1a6b58", "openstack/network/v2/security_group.py": "1fde1184e845682d237ee89235ec87f554232433a6f74e6af362e0e8ba6904dc", "openstack/proxy.py": "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46", "openstack/resource.py": "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0"}

func securityGroupPythonQueryFields() map[string]string {
	return map[string]string{"any_tags": "tags-any", "description": "description", "fields": "fields", "id": "id", "is_shared": "shared", "limit": "limit", "marker": "marker", "name": "name", "not_any_tags": "not-tags-any", "not_tags": "not-tags", "project_id": "project_id", "revision_number": "revision_number", "sort_dir": "sort_dir", "sort_key": "sort_key", "stateful": "stateful", "tags": "tags", "tenant_id": "tenant_id"}
}
func securityGroupPythonBodyFields() map[string]pythonFilterField {
	list := "list"
	return map[string]pythonFilterField{"created_at": {Field: "created_at"}, "updated_at": {Field: "updated_at"}, "security_group_rules": {Field: "security_group_rules", ResponseType: &list}}
}
func securityGroupPythonFilterMetadataValid(m *pythonFilterManifest) bool {
	if m == nil || m.SchemaVersion != 1 || m.SourcePin != pythonFilterPin || m.Resource != securityGroupPythonResource || m.SDKPackage != "github.com/JSYoo5B/go-openstacksdk/"+securityGroupSDKPath || m.BasePath != "/security-groups" || m.Envelope != "security_groups" || m.Counts.CanonicalQuery != 17 || m.Counts.AcceptedQuery != 21 || m.Counts.LocalBody != 3 || !reflect.DeepEqual(m.Query, securityGroupPythonQueryFields()) || !reflect.DeepEqual(m.Body, securityGroupPythonBodyFields()) || len(m.QueryFormats) != 0 || len(m.URI) != 0 || m.UnknownFilters != "discard" || m.QueryCollision != "canonical_client_name_wins" || !reflect.DeepEqual(m.ClassBases, []string{"_base.NetworkResource", "_base.TagMixinNetwork"}) || !reflect.DeepEqual(m.MRO, []string{"openstack.network.v2.security_group.SecurityGroup", "openstack.network.v2._base.NetworkResource", "openstack.resource.Resource", "builtins.dict", "openstack.network.v2._base.TagMixinNetwork", "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol", "typing.Protocol"}) || len(m.Proof.Nodes) != 39 {
		return false
	}
	wanted := map[string]string{"NetworkResource": "openstack/network/v2/_base.py", "NetworkResource.revision_number": "openstack/network/v2/_base.py", "Proxy._list": "openstack/proxy.py", "Proxy.security_groups": "openstack/network/v2/_proxy.py", "QueryParameters.__init__": "openstack/resource.py", "QueryParameters._transpose": "openstack/resource.py", "QueryParameters._validate": "openstack/resource.py", "Resource": "openstack/resource.py", "Resource.__getattribute__": "openstack/resource.py", "Resource.__getitem__": "openstack/resource.py", "Resource.__init__": "openstack/resource.py", "Resource._alternate_id": "openstack/resource.py", "Resource._attributes_iterator": "openstack/resource.py", "Resource._collect_attrs": "openstack/resource.py", "Resource._get_next_link": "openstack/resource.py", "Resource._max_microversion": "openstack/resource.py", "Resource.id": "openstack/resource.py", "Resource.list": "openstack/resource.py", "Resource.name": "openstack/resource.py", "Resource.to_dict": "openstack/resource.py", "ResourceMixinProtocol": "openstack/resource.py", "SecurityGroup": "openstack/network/v2/security_group.py", "SecurityGroup._query_mapping": "openstack/network/v2/security_group.py", "SecurityGroup.created_at": "openstack/network/v2/security_group.py", "SecurityGroup.description": "openstack/network/v2/security_group.py", "SecurityGroup.is_shared": "openstack/network/v2/security_group.py", "SecurityGroup.name": "openstack/network/v2/security_group.py", "SecurityGroup.project_id": "openstack/network/v2/security_group.py", "SecurityGroup.security_group_rules": "openstack/network/v2/security_group.py", "SecurityGroup.stateful": "openstack/network/v2/security_group.py", "SecurityGroup.tenant_id": "openstack/network/v2/security_group.py", "SecurityGroup.updated_at": "openstack/network/v2/security_group.py", "TagMixin": "openstack/common/tag.py", "TagMixin._tag_query_parameters": "openstack/common/tag.py", "TagMixin.tags": "openstack/common/tag.py", "TagMixinNetwork": "openstack/network/v2/_base.py", "_BaseComponent.__get__": "openstack/fields.py", "_BaseComponent.__init__": "openstack/fields.py", "_convert_type": "openstack/fields.py"}
	for _, n := range m.Proof.Nodes {
		if p, ok := wanted[n.Symbol]; ok && p == n.Source {
			delete(wanted, n.Symbol)
		}
	}
	return len(wanted) == 0
}
func verifySecurityGroupPythonFilterManifest(source string, m *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, m, "SecurityGroup", securityGroupPythonResource, securityGroupFilterSourceHashes, securityGroupPythonFilterMetadataValid)
}
func loadSecurityGroupPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "SecurityGroup", securityGroupFilterManifestPath, verifySecurityGroupPythonFilterManifest)
}
