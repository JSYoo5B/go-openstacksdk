// Package v3 is the Keystone identity v3 service. Connection.IdentityV3 returns
// a Service whose fields wrap Gophercloud's identity v3 calls and add
// SDK-owned EndpointGroups (OS-EP-FILTER) and IdentityProviders,
// FederationProtocols and ServiceProviders (OS-FEDERATION).
//
// Projects, Users, Groups, Domains and Roles provide FindIdentity, which
// resolves a name or ID string: GET first, then after a 400, 403 or 404 a list
// with exact ID and name matching and duplicate checks. A missing result is
// nil, nil unless resource.WithIdentityFindIgnoreMissing(false) is given.
// Filters from resource.WithIdentityFindQuery apply to the GET and the fallback list;
// the automatic name hint applies only to the list.
//
// Python openstacksdk mappings and differences are recorded in
// api/sdk_reviews.json.
package v3
