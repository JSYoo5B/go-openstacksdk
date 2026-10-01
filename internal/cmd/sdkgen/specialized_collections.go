package main

// Bindings with incompatible native list/detail models or alternate endpoint
// shapes are implemented in their SDK packages, rather than inferred as CRUD.
var specializedCollections = map[string]collectionRecord{
	upstreamModule + "/openstack/orchestration/v1/stacks": {
		Package: "gophercloudsdk/orchestration/v1/stacks", Model: "StackResource", UpstreamModel: "RetrievedStack", Kind: "compound_identity", Find: true, Delete: true, Wait: true,
	},
	upstreamModule + "/openstack/sharedfilesystems/v2/shareaccessrules": {
		Package: "gophercloudsdk/sharedfilesystems/v2/shareaccessrules", Model: "AccessRule", UpstreamModel: "ShareAccess", Delete: true, Wait: true,
		Scope: "InShare", Parent: "gophercloudsdk/sharedfilesystems/v2/shares",
	},
	upstreamModule + "/openstack/compute/v2/quotasets": {
		Package: "gophercloudsdk/compute/v2/quotasets", Model: "QuotaResource", UpstreamModel: "QuotaSet", Kind: "singleton",
		Scope: "InProject", Parent: "gophercloudsdk/identity/v3/projects",
	},
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

// These resources are implemented directly against the pinned Python/Cyborg
// contracts. Source distinguishes them from inferred native Gophercloud models.
var sdkOwnedCollections = []collectionRecord{
	{Package: "gophercloudsdk/accelerator/v2/devices", Source: "sdk_owned", Model: "Device", Wait: true},
	{Package: "gophercloudsdk/accelerator/v2/deployables", Source: "sdk_owned", Model: "Deployable", Find: true},
}
