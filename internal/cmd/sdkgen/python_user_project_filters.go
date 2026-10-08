package main

import "reflect"

const userProjectPythonResource = "openstack.identity.v3.project.UserProject"
const userProjectFilterManifestPath = "api/openstacksdk/resources/identity/v3/user_project.json"

var userProjectFilterSourceHashes = map[string]string{
	"openstack/identity/v3/project.py": "34fe73646e82d303f6e0f06cc13c8102dd2e7f74e5b1321f1e07683f949f7a93",
	"openstack/identity/v3/_proxy.py":  "de6b2d2276f376954600246217e5c4fcaa58cff493b884c22f11da8e4e8f28de",
	"openstack/common/tag.py":          "c86f1d41fc1ed139366f163e4d5dfb7a37b15380dda05c3382c47dd7bafaa59e",
	"openstack/resource.py":            "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0",
	"openstack/fields.py":              "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd",
	"openstack/proxy.py":               "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46",
}

func userProjectPythonQueryFields() map[string]string {
	return map[string]string{"domain_id": "domain_id", "is_domain": "is_domain", "name": "name", "parent_id": "parent_id", "is_enabled": "enabled", "tags": "tags", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any", "limit": "limit", "marker": "marker"}
}

func userProjectPythonBodyFields() map[string]pythonFilterField {
	dict := "dict"
	return map[string]pythonFilterField{"id": {Field: "id"}, "description": {Field: "description"}, "options": {Field: "options", ResponseType: &dict}, "links": {Field: "links"}}
}

// This descriptor belongs to the manual SDK-owned UserProject record. It must
// not enable Project filters on Users.Resources or claim a native collection.
func userProjectPythonFilterMetadataValid(manifest *pythonFilterManifest) bool {
	if manifest == nil || manifest.SchemaVersion != 1 || manifest.SourcePin != pythonFilterPin || manifest.Resource != userProjectPythonResource || manifest.SDKPackage != "github.com/JSYoo5B/go-openstacksdk/identity/v3/users" || manifest.BasePath != "/users/%(user_id)s/projects" || manifest.Envelope != "projects" || manifest.Counts.CanonicalQuery != 11 || manifest.Counts.AcceptedQuery != 15 || manifest.Counts.LocalBody != 4 || !reflect.DeepEqual(manifest.Query, userProjectPythonQueryFields()) || !reflect.DeepEqual(manifest.Body, userProjectPythonBodyFields()) || len(manifest.QueryFormats) != 0 || !reflect.DeepEqual(manifest.URI, map[string]pythonFilterField{"user_id": {Field: "user_id"}}) || manifest.UnknownFilters != "discard" || manifest.QueryCollision != "canonical_client_name_wins" || !reflect.DeepEqual(manifest.ClassBases, []string{"Project"}) || !reflect.DeepEqual(manifest.MRO, []string{userProjectPythonResource, "openstack.identity.v3.project.Project", "openstack.resource.Resource", "builtins.dict", "openstack.common.tag.TagMixin", "openstack.resource.ResourceMixinProtocol", "typing.Protocol"}) || len(manifest.Proof.Nodes) != 30 {
		return false
	}
	if !reflect.DeepEqual(manifest.Reserved, []string{"allow_unknown_params", "base_path", "headers", "jmespath_filters", "max_items", "microversion", "paginated", "resource_type", "session"}) || !reflect.DeepEqual(manifest.SourceControls.ResourceList, []string{"session", "paginated", "base_path", "allow_unknown_params", "microversion", "headers", "max_items"}) || !reflect.DeepEqual(manifest.SourceControls.ProxyList, []string{"resource_type", "paginated", "base_path", "jmespath_filters"}) {
		return false
	}
	wanted := map[string]string{"UserProject": "openstack/identity/v3/project.py", "UserProject.user_id": "openstack/identity/v3/project.py", "Project._query_mapping": "openstack/identity/v3/project.py", "Project.options": "openstack/identity/v3/project.py", "Project.links": "openstack/identity/v3/project.py", "Resource._get_next_link": "openstack/resource.py", "Proxy.user_projects": "openstack/identity/v3/_proxy.py"}
	for _, node := range manifest.Proof.Nodes {
		if path, ok := wanted[node.Symbol]; ok && path == node.Source {
			delete(wanted, node.Symbol)
		}
	}
	return len(wanted) == 0
}

func verifyUserProjectPythonFilterManifest(source string, manifest *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, manifest, "UserProject", userProjectPythonResource, userProjectFilterSourceHashes, userProjectPythonFilterMetadataValid)
}

func loadUserProjectPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "UserProject", userProjectFilterManifestPath, verifyUserProjectPythonFilterManifest)
}
