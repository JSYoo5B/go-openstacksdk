// Package v2 is the Neutron network v2 service. Connection.NetworkV2 returns a
// Service whose fields expose the API packages below this directory.
//
// Networks, Subnets, Ports, Routers, SecurityGroups, SubnetPools, Trunks,
// QoSPolicies and SecurityAddressGroups provide FindIdentity, which tries GET
// first and, after a 400, 403 or 404, lists with exact ID and name matching and
// duplicate checks; a missing result is nil, nil unless
// resource.WithIdentityFindIgnoreMissing(false) is given.
//
// Selective reads such as GetWithFields translate Python attribute names to
// wire fields. Update APIs send UpdateOpts.RevisionNumber as If-Match:
// revision_number=N, DeleteWithOptions accepts
// request.WithNeutronDeleteRevision and request.WithNeutronDeleteIgnoreMissing,
// and a 412 is never retried without the condition.
//
// Resources.List and All accept resource.WithBodyFilter for QoS policy rules,
// address group addresses, subnet pool prefixes and network subnets, compared
// locally after the rows are read.
//
// Record methods (GetRecord, FindRecord, ListRecords, CreateRecord,
// UpdateRecord, DeleteRecord) follow the Python proxy shapes for RBAC policies,
// QoS rules, trunks, address groups and scopes, subnet pools, security groups
// and rules, port forwardings and the core resources. CreateWithAttributes and
// UpdateWithAttributes on networks, ports, subnets, routers, floating IPs and
// security groups accept canonical names, raw body fields and revision
// conditions; an empty update sends no request.
//
// Connection.NetworkProjectQuotas and CurrentNetworkProjectQuotas fix a project
// for quota calls. Python openstacksdk mappings and differences are recorded in
// api/sdk_reviews.json.
package v2
