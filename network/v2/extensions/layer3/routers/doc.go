// Package routers exposes Gophercloud v2.15.0's
// openstack/networking/v2/extensions/layer3/routers calls through API, with
// concrete options, the SDK's request call categories and typed results.
// Connection.NetworkV2 returns the network/v2 service, whose Routers field
// provides this API.
//
// `Routers.Resources.List/All` classifies `resource.WithFilter`/`WithFilters`
// into 18 queries, 24 accepted names and ten local Body fields. It maps three
// boolean queries and tag aliases; name/status/id are server conditions.
// Gateway/routes/availability zones/timestamps/tenant ID use original JSON;
// enable_ndp_proxy uses nullable truthiness, and evpn_vni/revision use exact
// integers. Python revision_number and explicit Body aliases select raw
// revision, without falling back to the revision_number read by native
// RevisionNumber. Native nested/revision/timestamp decoding happens first for
// the whole page. Existing WithName/WithStatus, final wire status, native typed
// List/FindIdentity/Get/Delete/interface mutations remain available.
// `Network(ctx).API.Routers.Resources` and `NetworkV2(ctx).Routers.Resources`
// share the cached client.
package routers
