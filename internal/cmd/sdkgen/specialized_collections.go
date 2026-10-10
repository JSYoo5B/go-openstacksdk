package main

// Bindings with incompatible native list/detail models or alternate endpoint
// shapes are implemented in their SDK packages, rather than inferred as CRUD.
var specializedCollections = map[string]collectionRecord{
	upstreamModule + "/openstack/orchestration/v1/stacks": {
		Package: "github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks", Model: "StackResource", UpstreamModel: "RetrievedStack", Kind: "compound_identity", Find: true, Delete: true, Wait: true,
	},
	upstreamModule + "/openstack/orchestration/v1/stackresources": {
		Package: "github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources", Model: "ResourceView", UpstreamModel: "Resource", Kind: "compound_child", Find: true, Wait: true,
		Scope: "InStack", Parent: "github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks",
	},
	upstreamModule + "/openstack/orchestration/v1/stackevents": {
		Package: "github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents", Model: "EventResource", UpstreamModel: "Event", Kind: "event_log",
		Scope: "InStack", Parent: "github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks",
	},
	upstreamModule + "/openstack/sharedfilesystems/v2/shareaccessrules": {
		Package: "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules", Model: "AccessRule", UpstreamModel: "ShareAccess", Delete: true, Wait: true,
		Scope: "InShare", Parent: "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares",
	},
	upstreamModule + "/openstack/compute/v2/quotasets": {
		Package: "github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets", Model: "QuotaResource", UpstreamModel: "QuotaSet", Kind: "singleton",
		Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects",
	},
	upstreamModule + "/openstack/blockstorage/v3/quotasets": {
		Package: "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets", Model: "QuotaResource", UpstreamModel: "QuotaSet", Kind: "singleton",
		Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects",
	},
	upstreamModule + "/openstack/networking/v2/extensions/quotas": {
		Package: "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas", Model: "QuotaResource", UpstreamModel: "Quota", Kind: "singleton",
		Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects",
	},
	upstreamModule + "/openstack/loadbalancer/v2/quotas": {
		Package: "github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas", Model: "QuotaResource", UpstreamModel: "Quota", Kind: "singleton",
		Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects",
	},
	upstreamModule + "/openstack/dns/v2/quotas": {
		Package: "github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas", Model: "QuotaResource", UpstreamModel: "Quota", Kind: "singleton",
		Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects",
	},
	upstreamModule + "/openstack/compute/v2/limits": {
		Package: "github.com/JSYoo5B/go-openstacksdk/compute/v2/limits", Model: "LimitsResource", UpstreamModel: "Limits", Kind: "project_limits",
		Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects",
	},
	upstreamModule + "/openstack/blockstorage/v3/limits": {
		Package: "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/limits", Model: "LimitsResource", UpstreamModel: "Limits", Kind: "project_limits",
		Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects",
	},
	upstreamModule + "/openstack/containerinfra/v1/quotas": {
		Package: "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas", Model: "QuotaResource", UpstreamModel: "Quotas", Kind: "project_resource_quota",
		Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects",
	},
	upstreamModule + "/openstack/objectstorage/v1/containers": {
		Package: "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers", Model: "ContainerResource", UpstreamModel: "Container", Find: true, Delete: true,
	},
	upstreamModule + "/openstack/objectstorage/v1/objects": {
		Package: "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects", Model: "ObjectResource", UpstreamModel: "Object", Find: true, Delete: true,
		Scope: "InContainer", Parent: "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers",
	},
	upstreamModule + "/openstack/compute/v2/instanceactions": {
		Package: "github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions", Model: "ActionResource", UpstreamModel: "InstanceAction",
		Scope: "InServer", Parent: "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers",
	},
	upstreamModule + "/openstack/compute/v2/tags": {
		Package: "github.com/JSYoo5B/go-openstacksdk/compute/v2/tags", Model: "string", Kind: "string_set", Delete: true,
		Scope: "InServer", Parent: "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers",
	},
	upstreamModule + "/openstack/db/v1/databases": {
		Package: "github.com/JSYoo5B/go-openstacksdk/db/v1/databases", Model: "Database", Find: true, Delete: true,
		Scope: "InInstance", Parent: "github.com/JSYoo5B/go-openstacksdk/db/v1/instances",
	},
	upstreamModule + "/openstack/db/v1/users": {
		Package: "github.com/JSYoo5B/go-openstacksdk/db/v1/users", Model: "UserResource", UpstreamModel: "User", Find: true, Delete: true,
		Scope: "InInstance", Parent: "github.com/JSYoo5B/go-openstacksdk/db/v1/instances",
	},
}

// These resources are implemented directly against documented service and
// pinned Python contracts. Source distinguishes them from native models.
var sdkOwnedCollections = []collectionRecord{
	{Package: "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces", Source: "sdk_owned", Model: "Namespace", Kind: "named_resource", Delete: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects", Source: "sdk_owned", Model: "Object", Kind: "scoped_named_resource", Delete: true, Scope: "InNamespace", Parent: "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties", Source: "sdk_owned", Model: "Property", Kind: "scoped_definition", Delete: true, Scope: "InNamespace", Parent: "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags", Source: "sdk_owned", Model: "Tag", Kind: "scoped_tag_resource", Delete: true, Scope: "InNamespace", Parent: "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes", Source: "sdk_owned", Model: "ResourceType", Kind: "catalog_and_association", Delete: true, Scope: "InNamespace", Parent: "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo", Source: "sdk_owned", Model: "Store", Kind: "list_only"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo", Source: "sdk_owned", Model: "ImportInfo", Kind: "service_info"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo", Source: "sdk_owned", Model: "UsageInfo", Kind: "service_info"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores", Source: "sdk_owned", Model: "SecretStore", Kind: "store_defaults"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas", Source: "sdk_owned", Model: "Quota", Kind: "effective_project_quota", Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretconsumers", Source: "sdk_owned", Model: "Consumer", Kind: "secret_consumer", Scope: "InSecret", Parent: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretacls", Source: "sdk_owned", Model: "SecretACL", Kind: "secret_acl", Scope: "InSecret", Parent: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions", Source: "sdk_owned", Model: "Subscription", Kind: "queue_subscription", Scope: "InQueue", Parent: "github.com/JSYoo5B/go-openstacksdk/messaging/v2/queues"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotasets", Source: "sdk_owned", Model: "QuotaResource", Kind: "singleton", Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotaclasssets", Source: "sdk_owned", Model: "QuotaClassResource", Kind: "named_singleton", Scope: "InClass"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/accelerator/v2/devices", Source: "sdk_owned", Model: "Device", Wait: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deployables", Source: "sdk_owned", Model: "Deployable", Find: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles", Source: "sdk_owned", Model: "DeviceProfile", Find: true, Delete: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/accelerator/v2/attributes", Source: "sdk_owned", Model: "Attribute", Delete: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests", Source: "sdk_owned", Model: "AcceleratorRequest", Delete: true, Wait: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments", Source: "sdk_owned", Model: "Segment", Find: true, Delete: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/instanceha/v1/hosts", Source: "sdk_owned", Model: "Host", Find: true, Delete: true, Scope: "InSegment", Parent: "github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/instanceha/v1/notifications", Source: "sdk_owned", Model: "Notification", Wait: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/instanceha/v1/vmoves", Source: "sdk_owned", Model: "VMove", Wait: true, Scope: "InNotification", Parent: "github.com/JSYoo5B/go-openstacksdk/instanceha/v1/notifications"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/buildinfo", Source: "sdk_owned", Model: "BuildInfo", Kind: "service_info"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes", Source: "sdk_owned", Model: "ProfileType", Find: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes", Source: "sdk_owned", Model: "PolicyType", Find: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles", Source: "sdk_owned", Model: "Profile", Find: true, Delete: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies", Source: "sdk_owned", Model: "Policy", Find: true, Delete: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters", Source: "sdk_owned", Model: "Cluster", Kind: "async_resource", Find: true, Wait: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes", Source: "sdk_owned", Model: "Node", Kind: "async_resource", Find: true, Wait: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers", Source: "sdk_owned", Model: "Receiver", Find: true, Delete: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions", Source: "sdk_owned", Model: "Action", Find: true, Wait: true},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/events", Source: "sdk_owned", Model: "Event"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/services", Source: "sdk_owned", Model: "Service", Kind: "list_only"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies", Source: "sdk_owned", Model: "ClusterPolicy", Kind: "policy_binding", Find: true, Scope: "InCluster", Parent: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"},
	{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterattributes", Source: "sdk_owned", Model: "ClusterAttribute", Kind: "scoped_list_only", Scope: "InCluster", Parent: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"},
}
