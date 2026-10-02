package main

import "reflect"

const addressGroupPythonResource = "openstack.network.v2.address_group.AddressGroup"
const addressGroupFilterManifestPath = "api/openstacksdk/resources/network/v2/address_group.json"

var addressGroupFilterSourceHashes = map[string]string{
	"openstack/network/v2/address_group.py": "ba4bacb24761eb2b9e4dad0ac3f664e65c0ecb231b13813020cc5eac5f6ec54c",
	"openstack/network/v2/_proxy.py":        "e409ee081f8b9c44903a1b81a4d9f9590c59f9f947aa686f960a5171ab1a6b58",
	"openstack/resource.py":                 "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0",
	"openstack/fields.py":                   "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd",
	"openstack/proxy.py":                    "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46",
}

func addressGroupPythonQueryFields() map[string]string {
	return map[string]string{"limit": "limit", "marker": "marker", "fields": "fields", "sort_key": "sort_key", "sort_dir": "sort_dir", "name": "name", "description": "description", "project_id": "project_id"}
}

func addressGroupPythonBodyFields() map[string]pythonFilterField {
	list := "list"
	return map[string]pythonFilterField{"id": {Field: "id"}, "tenant_id": {Field: "tenant_id"}, "addresses": {Field: "addresses", ResponseType: &list}}
}

func addressGroupPythonFilterMetadataValid(manifest *pythonFilterManifest) bool {
	if manifest == nil || manifest.SchemaVersion != 1 || manifest.SourcePin != pythonFilterPin || manifest.Resource != addressGroupPythonResource || manifest.SDKPackage != "gophercloudsdk/network/v2/extensions/security/addressgroups" || manifest.BasePath != "/address-groups" || manifest.Envelope != "address_groups" || manifest.Counts.CanonicalQuery != 8 || manifest.Counts.AcceptedQuery != 8 || manifest.Counts.LocalBody != 3 || !reflect.DeepEqual(manifest.Query, addressGroupPythonQueryFields()) || !reflect.DeepEqual(manifest.Body, addressGroupPythonBodyFields()) || len(manifest.QueryFormats) != 0 || len(manifest.URI) != 0 || manifest.UnknownFilters != "discard" || manifest.QueryCollision != "canonical_client_name_wins" || !reflect.DeepEqual(manifest.ClassBases, []string{"resource.Resource"}) || !reflect.DeepEqual(manifest.MRO, []string{addressGroupPythonResource, "openstack.resource.Resource", "builtins.dict"}) || len(manifest.Proof.Nodes) != 16 {
		return false
	}
	// project_id's tenant_id fallback is a descriptor declaration, not a query
	// alias. The two local attributes id/tenant_id retain their own raw fields.
	wanted := map[string]string{"AddressGroup.project_id": "openstack/network/v2/address_group.py", "AddressGroup.tenant_id": "openstack/network/v2/address_group.py", "AddressGroup.addresses": "openstack/network/v2/address_group.py", "Resource.id": "openstack/resource.py", "Resource.__getattribute__": "openstack/resource.py", "QueryParameters.__init__": "openstack/resource.py"}
	for _, node := range manifest.Proof.Nodes {
		if path, ok := wanted[node.Symbol]; ok && path == node.Source {
			delete(wanted, node.Symbol)
		}
	}
	return len(wanted) == 0
}

func verifyAddressGroupPythonFilterManifest(source string, manifest *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, manifest, "AddressGroup", addressGroupPythonResource, addressGroupFilterSourceHashes, addressGroupPythonFilterMetadataValid)
}

func loadAddressGroupPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "AddressGroup", addressGroupFilterManifestPath, verifyAddressGroupPythonFilterManifest)
}
