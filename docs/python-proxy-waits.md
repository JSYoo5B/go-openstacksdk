# Proxy Waits for Neutron, Keystone, Placement, Inspector and Cinder

This guide describes the Go equivalents of the generic `Proxy.wait_for_status`
and `Proxy.wait_for_delete` helpers of the pinned openstacksdk
`network/v2`, `identity/v3`, `placement/v1`, `baremetal_introspection/v1`
and `block_storage/v3` proxies (`openstack/resource.py` `wait_for_status`
and `wait_for_delete`). Compute, Cinder Volume/Snapshot and Glance service
defaults are described in [service waits](service-waits.md).

## Service functions

| Service | Status wait | Deletion wait |
| --- | --- | --- |
| Neutron | `network.WaitForStatus` | `network.WaitForDelete` |
| Keystone v3 | `identity.WaitForStatus` | `identity.WaitForDelete` |
| Placement | `placement/v1.WaitForStatus` | `placement/v1.WaitForDelete` |
| Inspector | `baremetalintrospection/v1.WaitForStatus` | `baremetalintrospection/v1.WaitForDelete` |
| Cinder v3 | `blockstorage.WaitForState` | `blockstorage.WaitForDelete` |

Each function accepts any SDK `*resource.Collection[T]`, a `resource.Ref` and
`resource.WaitOption` values. SDK defaults are applied first and caller
options last.

- Status waits require a nonempty target, compare case-insensitively, fail on
  an exact case-insensitive `ERROR` (Cinder: `error`), poll every two seconds
  and have no SDK timeout. `resource.WithFailureStates`,
  `resource.WithPollInterval`, `resource.WithTimeout` and
  `resource.WithUnlimitedWait` replace these defaults.
- The Neutron, Keystone, Placement and Inspector functions select the model's
  exported string `status` attribute when the model declares one. A model
  without it fails with `resource.ErrUnsupported` before any HTTP request,
  as Python raises `AttributeError` for its default attribute. Select another
  exported string field with `resource.WithStatusAttribute`; numeric and
  boolean fields are unsupported, as Python's `lower()` call on them also
  fails. Cinder collections use their binding status accessor. Records kept
  as raw JSON select Python attribute names instead; see
  [Raw JSON records](#raw-json-records).
- Deletion waits poll every two seconds for up to 120 seconds. Only GET
  requests are sent; a waiter never sends DELETE, abort or any mutation. An
  actual 404 or a successful nil observation completes. A model with a status
  attribute also completes on a case-insensitive `deleted` value. HTTP errors
  other than 404 (for example 403), transport, decoder and context errors end
  the wait with that error.
- `resource.WithProgressCallback` reports an exported integer `progress`
  field or zero after each nonterminal observation.

## Wait collections

Generated packages expose their collection as the `Resources` field. SDK-owned
packages and parent-scoped resources expose a method returning a collection
bound to the direct member GET. These method collections accept only
`resource.ID` references; listing, name lookup and deletion stay on the
package API.

Neutron:

- `localips.API.Resources()`, `localipassociations.API.Resources(localIPID)`
- `networksegmentranges.API.Resources()`
- `serviceprofiles.API.Resources()`, `flavors.API.Resources()`
- `metering/labels.API.Resources()`, `metering/rules.API.Resources()`
- `taas/tapflows.API.Resources()`, `taas/tapservices.API.Resources()`
  (both declare `status`)
- `sfc/portpairs`, `sfc/portpairgroups`, `sfc/portchains`,
  `sfc/flowclassifiers`, `sfc/servicegraphs`: `API.Resources()`
- `security/defaultrules.API.Resources()`,
  `security/defaultstatefulness.API.Resources()`
- `layer3/ndpproxies.API.Resources()`,
  `layer3/conntrackhelpers.API.Resources(routerID)`
- `quotas.API.Resources()` (project ID), `networkipavailabilities.API.Resources()`
  (network ID), `qos/ruletypes.API.Resources()` (rule type name),
  `extensions.API.Resources()` (alias)
- `qos/rules.API.MinimumPacketRateResources(policyID)` and
  `PacketRateLimitResources(policyID)`
- `bgpvpns.API.NetworkAssociationResources`, `PortAssociationResources` and
  `RouterAssociationResources` (BGP VPN ID)

Keystone v3: `identityproviders.API.Resources()`,
`serviceproviders.API.Resources()`,
`federationprotocols.API.Resources(identityProviderID)`,
`federation.API.MappingResources()`,
`applicationcredentials.API.AccessRuleResources(userID)`,
`domains.API.ConfigResources()` (domain ID) and
`endpointgroups.API.Resources()`. Application credentials use
`applicationcredentials.API.InUser`.

Placement: `allocations.API.Resources()` (consumer UUID; `Allocation`
carries `ConsumerUUID`) and `resourceproviders.API.TraitResources()` (provider
ID), in addition to the existing provider, resource class, inventory and trait
collections.

Inspector: `introspection.API.Resources` and `rules.API.Resources()`.

Cinder v3: `groups`, `groupsnapshots`, `consistencygroups` and
`consistencygroupsnapshots` `API.Resources()` read the record `status`
attribute; `clusters.API.Resources()` (cluster name, reads `Status`),
`defaulttypes.API.Resources()` (project ID), `grouptypes.API.Resources()`,
`messages.API.Resources()`, `capabilities.API.Resources()` (backend host) and
`quotasets.API.QuotaClassResources()` (class name).

## Service observations

- Neutron `GET auto-allocated-topology/{project_id}` may provision a topology,
  so Go provides no waiter for it: polling it would be a mutation.
- Placement answers GET for a consumer without allocations with an empty
  allocation set, and Keystone answers 404 for a domain without configuration.
  A deletion wait reports only what the service returns, so an allocation
  wait normally reaches its deadline, as in Python.
- The Inspector has no member DELETE route and keeps finished or aborted
  introspection status readable; deletion waits on it usually reach their
  deadline. `introspection.API.WaitUntilFinished` remains the finished-flag
  waiter.

## Raw JSON records

Some records keep their declared Resource view as raw JSON instead of
exported Go fields: Neutron segment ranges, default security group rules,
default statefulness, NDP proxies, conntrack helpers and packet-rate rules;
Keystone endpoint groups and domain configs; Cinder group, group snapshot,
consistency group, consistency group snapshot, message, capabilities and
quota class records. Their collections set `resource.Adapter.WaitAttributes`,
so `resource.WithStatusAttribute` selects a key of the record's `Resource`
view:

- The name is the exact Python attribute name. Where the wire key differs, the
  view uses the attribute name: a default security group rule is selected with
  `ether_type`, not `ethertype`. Names are case-sensitive.
- A JSON string value is compared like a typed field: case-insensitive target,
  the same default (`ERROR`, Cinder `error`) and caller failure states, and
  `deleted` completion for deletion waits when the attribute is selected there.
- An attribute the view does not carry, a null value and a numeric, boolean,
  list or object value fail with `resource.ErrUnsupported`. The value is read
  from each observation, so this error arrives after the first GET rather than
  before any request. Python keeps polling a `None` value until its timeout.
- Classes that allow unknown Body attributes (for example NetworkSegmentRange
  and NDPProxy) keep undeclared response members in the view, so they can be
  selected; ConntrackHelper and other classes discard them.
- URI attributes (`router_id`, `qos_policy_id`, a domain config's
  `domain_id`) are kept outside the view and cannot be selected.
- Status is selected by default only where the binding declares it (Cinder
  group-family records); without `resource.WithStatusAttribute` the others
  fail with `resource.ErrUnsupported` before any request. Progress callbacks
  report zero.

```go
rules := defaultrules.New(client).Resources()
rule, err := network.WaitForStatus(ctx, rules, resource.ID(ruleID), "IPv6",
	resource.WithStatusAttribute("ether_type"))
```

## Differences from Python

- Go accepts an explicit `resource.Ref` and a typed collection and always
  starts with a fresh GET. Python accepts an existing Resource and returns it
  without HTTP when its cached attribute already equals the target.
- `WaitForDelete` returns only an error; Python returns the original or the
  fetched Resource.
- Python raises `ResourceTimeout`, `ResourceFailure` and `AttributeError`; Go
  returns the context deadline, `resource.ErrFailedState` and
  `resource.ErrUnsupported`.
- Go selects exported Go field names or JSON tags of typed models and Python
  attribute names of raw JSON record views; Python selects any attribute,
  including URI attributes.

Tests: `internal/servicewait/proxy_test.go`,
`resource/wait_raw_attributes_test.go`,
`api/raw_record_wait_attributes_test.go`,
`api/network_proxy_waits_test.go`, `api/identity_proxy_waits_test.go`,
`api/placement_proxy_waits_test.go`, `api/introspection_proxy_waits_test.go`,
`api/cinder_proxy_waits_test.go`.
