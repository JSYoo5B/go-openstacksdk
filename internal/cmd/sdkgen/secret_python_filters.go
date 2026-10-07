package main

import (
	"go/types"
	"reflect"
)

const secretPythonResource = "openstack.key_manager.v1.secret.Secret"
const secretFilterManifestPath = "api/openstacksdk/resources/key_manager/v1/secret.json"
const secretPythonFormatter = "openstack.key_manager.v1._format.HREFToUUID"

var secretFilterSourceHashes = map[string]string{
	"openstack/key_manager/v1/secret.py":  "1d2ba42bbf41014a51f4d9f859b0077098c462925ebed24b8a3e59e299d2b9ed",
	"openstack/resource.py":               "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0",
	"openstack/fields.py":                 "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd",
	"openstack/proxy.py":                  "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46",
	"openstack/key_manager/v1/_proxy.py":  "611b30f4762cf926cd260996b3206f90fb0ba8918d41cd63dd2c18a68bb4c228",
	"openstack/key_manager/v1/_format.py": "7bd671a212db94a118b271fcf4ccf0f0967b9e56f52c754b5a315aa40592d83d",
	"openstack/format.py":                 "dd36b5a20158ffd0edf17bf6ae92432370e1495d4f61a12bee6b0ccb0c7aebcd",
}

func secretPythonQueryFields() map[string]string {
	return map[string]string{"acl_only": "acl_only", "algorithm": "alg", "bits": "bits", "created": "created", "expiration": "expiration", "limit": "limit", "marker": "marker", "mode": "mode", "name": "name", "secret_type": "secret_type", "sort": "sort", "updated": "updated"}
}

func secretPythonBodyFields() map[string]pythonFilterField {
	dict, formatter := "dict", secretPythonFormatter
	return map[string]pythonFilterField{
		"bit_length": {Field: "bit_length"}, "content_types": {Field: "content_types", ResponseType: &dict},
		"created_at": {Field: "created"}, "expires_at": {Field: "expiration"}, "updated_at": {Field: "updated"},
		"id": {Field: "id", ResponseAccessor: "resource_id"}, "payload": {Field: "payload"},
		"payload_content_encoding": {Field: "payload_content_encoding"}, "payload_content_type": {Field: "payload_content_type"},
		"secret_id":  {Field: "secret_ref", ResponseType: &formatter, Formatter: secretPythonFormatter},
		"secret_ref": {Field: "secret_ref"}, "status": {Field: "status"},
	}
}

func secretPythonFilterMetadataValid(manifest *pythonFilterManifest) bool {
	return manifest != nil && manifest.SchemaVersion == 1 && manifest.SourcePin == pythonFilterPin &&
		manifest.Resource == secretPythonResource && manifest.SDKPackage == "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secrets" && manifest.BasePath == "/secrets" && manifest.Envelope == "secrets" &&
		manifest.Counts.CanonicalQuery == 12 && manifest.Counts.AcceptedQuery == 13 && manifest.Counts.LocalBody == 12 &&
		reflect.DeepEqual(manifest.Query, secretPythonQueryFields()) && reflect.DeepEqual(manifest.Body, secretPythonBodyFields()) &&
		len(manifest.QueryFormats) == 0 && len(manifest.URI) == 0 && manifest.UnknownFilters == "discard" && manifest.QueryCollision == "canonical_client_name_wins" &&
		reflect.DeepEqual(manifest.ClassBases, []string{"resource.Resource"}) && reflect.DeepEqual(manifest.MRO, []string{secretPythonResource, "openstack.resource.Resource", "builtins.dict"})
}

func verifySecretPythonFilterManifest(source string, manifest *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, manifest, "Secret", secretPythonResource, secretFilterSourceHashes, secretPythonFilterMetadataValid)
}

func loadSecretPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "Secret", secretFilterManifestPath, verifySecretPythonFilterManifest)
}

// The Secret manifest is accepted only for the exact guarded native plan; this
// does not add Secret to the separate native FindIdentity capability table.
func secretSemanticPlan(pkg *types.Package, plan *collectionPlan) bool {
	_, ok := bodyFilterCollectionContract(pkg, plan, 0)
	return sdkPath(pkg.Path()) == "keymanager/v1/secrets" && ok
}
