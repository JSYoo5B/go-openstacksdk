// Package subnets exposes Gophercloud v2.15.0's openstack/networking/v2/subnets
// calls through API, with concrete options, the SDK's request call categories
// and typed results. Connection.NetworkV2 returns the network/v2 service, whose
// Subnets field provides this API.
//
// `Subnets.Resources` also supports nine declared local Body fields: allocation
// pools, DNS nameservers, host routes, service types, timestamps, prefix
// length, tenant ID and revision number. It compares original page rows and
// returns typed Subnet. prefix_length aliases prefixlen; whole-page native
// decoding happens first.
//
// `Subnets.Resources.List/All` classifies Python names into 24 server queries
// or nine local Body fields. The SDK handles 30 accepted names including wire
// aliases, bulk canonical precedence, final-value validation, semantic
// replacement/clear and JSON snapshots. Collisions with raw queries, explicit
// Body or page/name hints fail before HTTP; unknown semantic names are
// discarded. Subnet generation validates the pinned AST manifest against
// current Source. Full Resource/cache, inherited continuation/session, Proxy
// conflicting attrs and JMESPath remain separate.
package subnets
