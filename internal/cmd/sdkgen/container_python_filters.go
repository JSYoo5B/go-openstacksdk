package main

import "reflect"

const containerPythonResource = "openstack.key_manager.v1.container.Container"
const containerFilterManifestPath = "api/openstacksdk/resources/key_manager/v1/container.json"

var containerFilterSourceHashes = map[string]string{
	"openstack/key_manager/v1/container.py": "af3005424e14b8c7a825d99811c19ed0915e88567b8e9de52802f2b124455ea7",
	"openstack/resource.py":                 "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0",
	"openstack/fields.py":                   "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd",
	"openstack/proxy.py":                    "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46",
	"openstack/key_manager/v1/_proxy.py":    "611b30f4762cf926cd260996b3206f90fb0ba8918d41cd63dd2c18a68bb4c228",
	"openstack/key_manager/v1/_format.py":   "7bd671a212db94a118b271fcf4ccf0f0967b9e56f52c754b5a315aa40592d83d",
	"openstack/format.py":                   "dd36b5a20158ffd0edf17bf6ae92432370e1495d4f61a12bee6b0ccb0c7aebcd",
}

func containerPythonBodyFields() map[string]pythonFilterField {
	list, formatter := "list", secretPythonFormatter
	return map[string]pythonFilterField{
		"id": {Field: "id", ResponseAccessor: "resource_id"}, "name": {Field: "name"},
		"container_ref": {Field: "container_ref"},
		"container_id":  {Field: "container_ref", ResponseType: &formatter, Formatter: secretPythonFormatter},
		"created_at":    {Field: "created"}, "updated_at": {Field: "updated"},
		"secret_refs": {Field: "secret_refs", ResponseType: &list}, "consumers": {Field: "consumers", ResponseType: &list},
		"status": {Field: "status"}, "type": {Field: "type"},
	}
}

func containerPythonFilterMetadataValid(manifest *pythonFilterManifest) bool {
	if manifest == nil || manifest.SchemaVersion != 1 || manifest.SourcePin != pythonFilterPin ||
		manifest.Resource != containerPythonResource || manifest.SDKPackage != "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers" || manifest.BasePath != "/containers" || manifest.Envelope != "containers" ||
		manifest.Counts.CanonicalQuery != 2 || manifest.Counts.AcceptedQuery != 2 || manifest.Counts.LocalBody != 10 ||
		!reflect.DeepEqual(manifest.Query, map[string]string{"limit": "limit", "marker": "marker"}) || !reflect.DeepEqual(manifest.Body, containerPythonBodyFields()) ||
		len(manifest.QueryFormats) != 0 || len(manifest.URI) != 0 || manifest.UnknownFilters != "discard" || manifest.QueryCollision != "canonical_client_name_wins" ||
		!reflect.DeepEqual(manifest.ClassBases, []string{"resource.Resource"}) || !reflect.DeepEqual(manifest.MRO, []string{containerPythonResource, "openstack.resource.Resource", "builtins.dict"}) {
		return false
	}
	// Container does not declare a query mapping. The inherited mapping and its
	// pagination defaults must both have independent source anchors.
	inherited, defaults := false, false
	for _, anchor := range manifest.Proof.Nodes {
		if anchor.Source == "openstack/resource.py" {
			inherited = inherited || anchor.Symbol == "Resource._query_mapping"
			defaults = defaults || anchor.Symbol == "QueryParameters.__init__"
		}
	}
	return inherited && defaults
}

func verifyContainerPythonFilterManifest(source string, manifest *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, manifest, "Container", containerPythonResource, containerFilterSourceHashes, containerPythonFilterMetadataValid)
}

func loadContainerPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "Container", containerFilterManifestPath, verifyContainerPythonFilterManifest)
}
