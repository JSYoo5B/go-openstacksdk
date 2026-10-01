package main

// Bindings with incompatible native list/detail models or alternate endpoint
// shapes are implemented in their SDK packages, rather than inferred as CRUD.
var specializedCollections = map[string]collectionRecord{
	upstreamModule + "/openstack/objectstorage/v1/containers": {
		Package: "gophercloudsdk/objectstorage/v1/containers", Model: "ContainerResource", UpstreamModel: "Container", Find: true, Delete: true,
	},
	upstreamModule + "/openstack/objectstorage/v1/objects": {
		Package: "gophercloudsdk/objectstorage/v1/objects", Model: "ObjectResource", UpstreamModel: "Object", Find: true, Delete: true,
		Scope: "InContainer", Parent: "gophercloudsdk/objectstorage/v1/containers",
	},
	upstreamModule + "/openstack/compute/v2/instanceactions": {
		Package: "gophercloudsdk/compute/v2/instanceactions", Model: "ActionResource", UpstreamModel: "InstanceAction",
		Scope: "InServer", Parent: "gophercloudsdk/compute/v2/servers",
	},
	upstreamModule + "/openstack/compute/v2/tags": {
		Package: "gophercloudsdk/compute/v2/tags", Model: "string", Kind: "string_set", Delete: true,
		Scope: "InServer", Parent: "gophercloudsdk/compute/v2/servers",
	},
	upstreamModule + "/openstack/db/v1/databases": {
		Package: "gophercloudsdk/db/v1/databases", Model: "Database", Find: true, Delete: true,
		Scope: "InInstance", Parent: "gophercloudsdk/db/v1/instances",
	},
	upstreamModule + "/openstack/db/v1/users": {
		Package: "gophercloudsdk/db/v1/users", Model: "UserResource", UpstreamModel: "User", Find: true, Delete: true,
		Scope: "InInstance", Parent: "gophercloudsdk/db/v1/instances",
	},
}
