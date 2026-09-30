package main

// Swift's HEAD responses differ from its list records, and its names are the
// identifiers. These bindings are implemented in their SDK packages rather than
// inferred by combining incompatible upstream models.
var specializedCollections = map[string]collectionRecord{
	upstreamModule + "/openstack/objectstorage/v1/containers": {
		Package: "gophercloudsdk/objectstorage/v1/containers", Model: "ContainerResource", UpstreamModel: "Container", Find: true, Delete: true,
	},
	upstreamModule + "/openstack/objectstorage/v1/objects": {
		Package: "gophercloudsdk/objectstorage/v1/objects", Model: "ObjectResource", UpstreamModel: "Object", Find: true, Delete: true,
		Scope: "InContainer", Parent: "gophercloudsdk/objectstorage/v1/containers",
	},
}
