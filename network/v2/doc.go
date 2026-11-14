// Package v2 is the Network v2 service. Connection.NetworkV2 returns a Service
// whose fields expose the API packages below this directory; README.md gives an
// overview.
//
// `Networks/Subnets/Ports/Routers/SecurityGroups/SubnetPools/Trunks/QoSPolicies/SecurityAddressGroups.FindIdentity(ctx,
// identity, options...)` resolves name/ID strings automatically, with list
// fallback after GET400/403/404, exact ID/name and duplicate checks, and a
// default missing result of `nil, nil`. Select strict handling with
// `resource.WithIdentityFindIgnoreMissing(false)`. Caller
// `WithIdentityFindQuery` filters survive GET and fallback listing; automatic
// name hints apply only to listing. Concrete options control query/fallback.
//
// The 27 selective reads, including `GetWithFields(ctx, id, fields...)` in 24
// Neutron packages and BGP VPN `Get*AssociationWithFields(ctx, bgpvpnID, id,
// fields...)`, translate Python Body attribute names to wire fields. Seven
// `DeleteWithOptions` APIs provide revision conditions and default
// ignore-missing through
// `request.WithNeutronDeleteRevision`/`request.WithNeutronDeleteIgnoreMissing`.
//
// Nine Neutron Update APIs send `UpdateOpts.RevisionNumber` as `If-Match:
// revision_number=N`. Nil omits the condition; a pointer to zero explicitly
// requests revision 0. The final `WithUpdateOptions` value also applies. A 412
// never triggers an unconditional retry.
//
// `resource.WithBodyFilter`/`WithBodyFilters` compares QoSPolicies rules,
// SecurityAddressGroups addresses, SubnetPools prefixes and Networks subnets
// locally. The SDK selects fields, snapshots JSON and compares whole arrays;
// callers implement no builder/predicate. Network accepts subnet_ids as an SDK
// alias of subnets, rejecting both together before HTTP. The higher-level
// `network.Service.Networks` shares these filters. Nil/empty arrays and
// filtering after raw caps are distinct; raw wire queries are not automatically
// classified. Only ordinary `Resources.List/All` accepts these options, not
// native typed List, FindIdentity or other resources.
//
// Owned QoS reads cover minimum bandwidth, minimum packet rate, packet rate
// limits, and rule types. Security group lists, security rule lists, router
// reads, and RBAC mutations explain Python/Go defaults, concrete options,
// permissions, and response boundaries.
//
// `RBACPolicies.GetRecord/FindRecord/ListRecords`, QoS
// `Get/Find/ListBandwidthLimitRecord` and `DSCPMarkingRecord`, and
// `Trunks/AddressGroups.ListRecords` expose concrete function options, Source
// views, and actual Wire evidence. RBAC, bandwidth limits, DSCP, trunk lists,
// and address group lists cover defaults, parent scopes, paging, and
// permissions.
//
// Owned `AddressScopes`, `SubnetPools`, and `QoSPolicies` Record methods
// provide defaults, function options, Source fields, and lookup/list policies.
// Address scopes, subnet pools, and QoS policies cover Python/Go usage and
// permission/version boundaries.
//
// `SecurityGroups/SecurityRules/Trunks/AddressGroups.FindRecord` owns name/ID
// lookup, missing defaults, and service-specific descriptors. Security lookup,
// trunk/address lookup, and router interfaces/gateways describe Source/stock
// differences, owned responses, and retry protections.
//
// `Networks/Ports/Subnets.FindRecord` provides GET-first lookup with Source
// queries, descriptors, and defaults. Core lookup, router routes/gateways,
// trunk/address group actions, and seven tag APIs describe concrete function
// options and response/cache/permission branches.
//
// `PortForwarding.CreateRecord/UpdateRecord/GetRecord/ListRecords/FindRecord/DeleteRecord`
// provides floating-IP parent scopes, raw fields, and missing defaults.
//
// `Networks/Ports/Subnets/Routers/FloatingIPs/SecurityGroups.CreateWithAttributes/UpdateWithAttributes`
// handles canonical aliases, raw Body, If-Match, and revision options. Empty
// updates complete without HTTP; revision-only updates send an empty Body PUT.
// Networks, ports, subnets, routers, floating IPs, security groups, and
// security rules/bulk creation cover defaults, conditional headers, and native
// decoding differences. See also resource creation/default deletion and
// selected reads/deletion.
//
// `conn.NetworkProjectQuotas(ctx, project)` and
// `CurrentNetworkProjectQuotas(ctx)` expose Neutron quotas as fixed-project
// singletons.
//
// Python openstacksdk mappings and differences are recorded in
// api/sdk_reviews.json.
package v2
