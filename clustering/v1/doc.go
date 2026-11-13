// Package v1 is the Senlin clustering v1 service. Pinned Gophercloud has no
// Senlin binding, so this SDK implements it. Connection.ClusteringV1 returns a
// Service for service type clustering; an unversioned catalog endpoint gains
// /v1 and existing versions and prefixes are kept.
//
// Profiles, Policies, Clusters, Nodes and Receivers provide CRUD, listing,
// name lookup and FindIdentity, which tries GET first and then searches every
// page for an exact name or ID. Their Track and Load methods return tracked
// values whose Commit sends a PATCH with only the changed fields. Cluster and
// node actions return the accepted action reference without waiting for it.
// Fetchable resources expose WaitForStatus (unlimited, ERROR fails, two-second
// interval by default) and WaitForDelete (120 seconds by default).
//
// Typed List and All calls accept request.MaxItems, request.Paginated,
// request.Header and request.Microversion, and each package's WithListFilter
// compares known body fields locally after the rows are read. MaxItems caps
// raw rows before local filtering. ClusterPolicies apply the cap locally; the
// other lists also send it as a limit hint, and deployment behavior determines
// support for type catalog and Services hints.
//
// Calls check the microversion selected for the call: Validate and
// ClusterAttributes need 1.2, ReplaceNodes 1.3, ProfileTypes.Operations,
// cluster and node operations, receiver list user and RemoveNodes
// destroy_after_deletion 1.4, profile_only and recover check 1.6, Nodes.Adopt,
// Services.List and recover check_capacity 1.7, Actions.Cancel 1.12 and node
// tainted 1.13.
//
// Python openstacksdk mappings and differences are recorded in
// api/sdk_reviews.json.
package v1
