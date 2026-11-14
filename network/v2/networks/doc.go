// Package networks exposes Gophercloud v2.15.0's
// openstack/networking/v2/networks calls through API, with concrete options,
// the SDK's request call categories and typed results. Connection.NetworkV2
// returns the network/v2 service, whose Networks field provides this API.
//
// `Networks.Resources.List/All` and higher-level `Network(ctx).Networks`
// classify `resource.WithFilter`/`WithFilters` into 23 queries, 35 accepted
// names and 14 local Body fields. Provider/address-scope/boolean/tag aliases
// are mapped; name/status/id are server conditions. The SDK compares raw
// subnets/availability zones/segments/timestamps/extensions, truthiness of four
// booleans and exact conversion of two integers, returning native Network.
// Existing exact WithName and case-insensitive WithStatus remain distinct; both
// list paths preserve raw status query. Shared independent
// `Networks.ResourceAdapter()` metadata preserves the facade's singular network
// error label and exact ERROR wait. Native typed List/FindIdentity/Get/Delete
// are independent.
package networks
