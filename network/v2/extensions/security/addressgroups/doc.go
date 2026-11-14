// Package addressgroups exposes Gophercloud v2.15.0's
// openstack/networking/v2/extensions/security/addressgroups calls through API,
// with concrete options, the SDK's request call categories and typed results.
// Connection.NetworkV2 returns the network/v2 service, whose
// SecurityAddressGroups field provides this API.
//
// `SecurityAddressGroups.Resources.List/All` classifies
// `resource.WithFilter`/`WithFilters` into eight server queries and three local
// Body fields. name/project_id are server queries; id/tenant_id/addresses are
// compared against original rows after whole-page native decoding. Comparisons
// preserve tenant ID absent from the native model and null address-array
// elements; results remain typed AddressGroup. WithName hints/local comparison,
// raw status queries, native typed List and FindIdentity remain separate.
package addressgroups
