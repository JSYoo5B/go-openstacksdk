package main

import "reflect"

const orderPythonResource = "openstack.key_manager.v1.order.Order"
const orderFilterManifestPath = "api/openstacksdk/resources/key_manager/v1/order.json"

var orderFilterSourceHashes = map[string]string{
	"openstack/key_manager/v1/order.py":   "d8688733b91c74a06b7307f9ba6feffabf499020062efc55995002354440ceb0",
	"openstack/resource.py":               "a3dc108ff184cf2928513a4c1e33295b85a5e485bc30b104e5b7e030dfdeb2e0",
	"openstack/fields.py":                 "dc557f53c445bb20cbc4dfddf44fb7dd4e3080f97a3bfa087b82f16f242d55cd",
	"openstack/proxy.py":                  "1eff2aa3ff960c7086fc7f8d2722b3c7d3f72316f58b25505491319213869a46",
	"openstack/key_manager/v1/_proxy.py":  "611b30f4762cf926cd260996b3206f90fb0ba8918d41cd63dd2c18a68bb4c228",
	"openstack/key_manager/v1/_format.py": "7bd671a212db94a118b271fcf4ccf0f0967b9e56f52c754b5a315aa40592d83d",
	"openstack/format.py":                 "dd36b5a20158ffd0edf17bf6ae92432370e1495d4f61a12bee6b0ccb0c7aebcd",
}

func orderPythonBodyFields() map[string]pythonFilterField {
	dict, formatter := "dict", secretPythonFormatter
	return map[string]pythonFilterField{
		"id": {Field: "id", ResponseAccessor: "resource_id"}, "name": {Field: "name"},
		"created_at": {Field: "created"}, "updated_at": {Field: "updated"}, "creator_id": {Field: "creator_id"},
		"meta": {Field: "meta", ResponseType: &dict}, "order_ref": {Field: "order_ref"},
		"order_id":   {Field: "order_ref", ResponseType: &formatter, Formatter: secretPythonFormatter},
		"secret_ref": {Field: "secret_ref"},
		"secret_id":  {Field: "secret_ref", ResponseType: &formatter, Formatter: secretPythonFormatter},
		"status":     {Field: "status"}, "sub_status": {Field: "sub_status"}, "sub_status_message": {Field: "sub_status_message"}, "type": {Field: "type"},
	}
}

func orderPythonFilterMetadataValid(manifest *pythonFilterManifest) bool {
	if manifest == nil || manifest.SchemaVersion != 1 || manifest.SourcePin != pythonFilterPin ||
		manifest.Resource != orderPythonResource || manifest.SDKPackage != "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/orders" || manifest.BasePath != "/orders" || manifest.Envelope != "orders" ||
		manifest.Counts.CanonicalQuery != 2 || manifest.Counts.AcceptedQuery != 2 || manifest.Counts.LocalBody != 14 ||
		!reflect.DeepEqual(manifest.Query, map[string]string{"limit": "limit", "marker": "marker"}) || !reflect.DeepEqual(manifest.Body, orderPythonBodyFields()) ||
		len(manifest.QueryFormats) != 0 || len(manifest.URI) != 0 || manifest.UnknownFilters != "discard" || manifest.QueryCollision != "canonical_client_name_wins" ||
		!reflect.DeepEqual(manifest.ClassBases, []string{"resource.Resource"}) || !reflect.DeepEqual(manifest.MRO, []string{orderPythonResource, "openstack.resource.Resource", "builtins.dict"}) {
		return false
	}
	// The inherited query defaults and both distinct reference formatters are
	// separate contracts; neither formatter substitutes for the passive id.
	wanted := map[string]string{"Resource._query_mapping": "openstack/resource.py", "QueryParameters.__init__": "openstack/resource.py", "Order.order_id": "openstack/key_manager/v1/order.py", "Order.secret_id": "openstack/key_manager/v1/order.py", "Order.meta": "openstack/key_manager/v1/order.py"}
	for _, anchor := range manifest.Proof.Nodes {
		if source, ok := wanted[anchor.Symbol]; ok && source == anchor.Source {
			delete(wanted, anchor.Symbol)
		}
	}
	return len(wanted) == 0
}

func verifyOrderPythonFilterManifest(source string, manifest *pythonFilterManifest) error {
	return verifyKeyManagerPythonFilterManifest(source, manifest, "Order", orderPythonResource, orderFilterSourceHashes, orderPythonFilterMetadataValid)
}

func loadOrderPythonFilterManifest(root, source string) (*pythonFilterManifest, error) {
	return loadKeyManagerPythonFilterManifest(root, source, "Order", orderFilterManifestPath, verifyOrderPythonFilterManifest)
}
