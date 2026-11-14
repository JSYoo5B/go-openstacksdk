// Package policies exposes Gophercloud v2.15.0's
// openstack/networking/v2/extensions/qos/policies calls through API, with
// concrete options, the SDK's request call categories and typed results.
// Connection.NetworkV2 returns the network/v2 service, whose QoSPolicies field
// provides this API.
//
// `QoSPolicies.Resources.List/All` classifies
// `resource.WithFilter`/`WithFilters` into 15 server queries, 19 accepted names
// and two local Body fields. It maps is_shared/tag aliases with bulk canonical
// precedence. rules/deprecated tenant_id are compared against original rows;
// id/name/project_id/booleans/tags are server queries. Large rule numbers are
// compared raw while native results retain float64. WithName, raw status
// queries, typed List and FindIdentity remain separate.
package policies
