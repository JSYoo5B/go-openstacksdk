package main

import "reflect"

const qosPolicyPythonResource = "openstack.network.v2.qos_policy.QoSPolicy"
const qosPolicyFilterManifestPath = "api/openstacksdk/resources/network/v2/qos_policy.json"

var qosPolicyFilterSourceHashes = map[string]string{
	"openstack/network/v2/qos_policy.py": "d82e98910ae942c725542fbc9ddd59b088fb3f60e3cf4e6631b72e2384e45dd3",
	"openstack/network/v2/_proxy.py":     "e409ee081f8b9c44903a1b81a4d9f9590c59f9f947aa686f960a5171ab1a6b58",
	"openstack/resource.py":              "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0",
	"openstack/fields.py":                "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd",
	"openstack/proxy.py":                 "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46",
	"openstack/common/tag.py":            "c86f1d41fc1ed139366f163e4d5dfb7a37b15380dda05c3382c47dd7bafaa59e",
}

func qosPolicyPythonQueryFields() map[string]string {
	return map[string]string{"limit": "limit", "marker": "marker", "fields": "fields", "name": "name", "description": "description", "id": "id", "is_default": "is_default", "project_id": "project_id", "sort_key": "sort_key", "sort_dir": "sort_dir", "is_shared": "shared", "tags": "tags", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any"}
}

func qosPolicyPythonBodyFields() map[string]pythonFilterField {
	list := "list"
	return map[string]pythonFilterField{"rules": {Field: "rules", ResponseType: &list}, "tenant_id": {Field: "tenant_id"}}
}

func qosPolicyPythonFilterMetadataValid(manifest *pythonFilterManifest) bool {
	if manifest == nil || manifest.SchemaVersion != 1 || manifest.SourcePin != pythonFilterPin || manifest.Resource != qosPolicyPythonResource || manifest.SDKPackage != "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies" || manifest.BasePath != "/qos/policies" || manifest.Envelope != "policies" || manifest.Counts.CanonicalQuery != 15 || manifest.Counts.AcceptedQuery != 19 || manifest.Counts.LocalBody != 2 || !reflect.DeepEqual(manifest.Query, qosPolicyPythonQueryFields()) || !reflect.DeepEqual(manifest.Body, qosPolicyPythonBodyFields()) || len(manifest.QueryFormats) != 0 || len(manifest.URI) != 0 || manifest.UnknownFilters != "discard" || manifest.QueryCollision != "canonical_client_name_wins" || !reflect.DeepEqual(manifest.ClassBases, []string{"resource.Resource", "tag.TagMixin"}) || !reflect.DeepEqual(manifest.MRO, []string{qosPolicyPythonResource, "openstack.resource.Resource", "builtins.dict", "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol", "typing.Protocol"}) || len(manifest.Proof.Nodes) != 18 {
		return false
	}
	wanted := map[string]string{"QoSPolicy.project_id": "openstack/network/v2/qos_policy.py", "QoSPolicy.tenant_id": "openstack/network/v2/qos_policy.py", "QoSPolicy.rules": "openstack/network/v2/qos_policy.py", "TagMixin._tag_query_parameters": "openstack/common/tag.py", "TagMixin.tags": "openstack/common/tag.py", "Resource.id": "openstack/resource.py", "Resource.__getattribute__": "openstack/resource.py", "QueryParameters.__init__": "openstack/resource.py"}
	for _, node := range manifest.Proof.Nodes {
		if path, ok := wanted[node.Symbol]; ok && path == node.Source {
			delete(wanted, node.Symbol)
		}
	}
	return len(wanted) == 0
}

func verifyQoSPolicyPythonFilterManifest(source string, manifest *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, manifest, "QoSPolicy", qosPolicyPythonResource, qosPolicyFilterSourceHashes, qosPolicyPythonFilterMetadataValid)
}

func loadQoSPolicyPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "QoSPolicy", qosPolicyFilterManifestPath, verifyQoSPolicyPythonFilterManifest)
}
