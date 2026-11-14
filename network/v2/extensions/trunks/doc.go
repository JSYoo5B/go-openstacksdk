// Package trunks exposes Gophercloud v2.15.0's
// openstack/networking/v2/extensions/trunks calls through API, with concrete
// options, the SDK's request call categories and typed results.
// Connection.NetworkV2 returns the network/v2 service, whose Trunks field
// provides this API.
//
// `Trunks.Resources.List/All` classifies `resource.WithFilter`/`WithFilters`
// into 14 queries, 18 accepted names and two raw-JSON local Body fields.
// Admin-state/tag aliases are mapped; id/tenant_id are raw conditions, while
// project_id/sub_ports/name/status are server conditions. A project response
// alias does not turn tenant Body into a query; native timestamp/revision are
// not semantic names. Whole-page Trunk/Subport decoding, RFC3339 timestamps and
// inherited links.next traversal remain. WithName compares exactly; WithStatus
// compares the final status query case-insensitively; raw status alone adds no
// local condition. Native typed List/FindIdentity/Get/Delete/subport mutations
// remain available. Both Network service paths share the cached client.
package trunks
