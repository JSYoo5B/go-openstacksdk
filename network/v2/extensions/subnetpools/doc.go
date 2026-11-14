// Package subnetpools exposes Gophercloud v2.15.0's
// openstack/networking/v2/extensions/subnetpools calls through API, with
// concrete options, the SDK's request call categories and typed results.
// Connection.NetworkV2 returns the network/v2 service, whose SubnetPools field
// provides this API.
//
// `SubnetPools.Resources.List/All` classifies
// `resource.WithFilter`/`WithFilters` into 16 queries, 20 accepted names and
// ten local Body fields. It maps is_shared/tag aliases and compares raw id,
// tenant ID, two timestamps, prefix arrays and five integer attributes. Python
// prefix-length names map to wire fields with exact-integer policy. Whole-page
// native required prefix-length/quota/revision/timestamp decoding happens
// first; results remain typed SubnetPool. Raw id/status queries, WithName,
// typed List/FindIdentity and prefix mutations remain separate.
package subnetpools
