# Network list filters

`Networks.Resources.List/All` classify `resource.WithFilter` and
`resource.WithFilters` using the pinned OpenStackSDK Network declarations.
The SDK sends declared query attributes to Neutron and compares fourteen local
Body attributes against original JSON rows after native page decoding. It
returns the existing native `*networks.Network` model. Applications supply
owned options rather than request builders or callback predicates.

Python:

```python
networks = list(conn.network.networks(
    is_shared=False,
    fields=["id", "name", "subnets", "is_default", "mtu"],
    subnet_ids=["subnet-one"],
    is_default=True,
    mtu=1500,
    max_items=20,
))
```

Go:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func listNetworks(ctx context.Context, client *gophercloud.ServiceClient) ([]*networks.Network, error) {
    return networks.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "is_shared": false,
            "fields": []string{"id", "name", "subnets", "is_default", "mtu"},
            "subnet_ids": []string{"subnet-one"},
            "is_default": true,
            "mtu": 1500,
        }),
        resource.WithMaxItems(20),
    )
}
```

Include each field needed by a local condition when using `fields`.
`subnet_ids` compares the complete ordered array. It does not infer membership,
normalize identifiers or refill the cap with matching rows. Local filters can
use extension fields absent from the native model, such as `is_default` and
`mtu`; those values are used for comparison without adding model fields.

Python:

```python
networks = list(conn.network.networks(
    name="server-condition",
    is_vlan_transparent=False,
    created_at="2025-01-02T03:04:05+00:00",
    paginated=False,
))
```

Go:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func firstPageNetworks(ctx context.Context, client *gophercloud.ServiceClient) ([]*networks.Network, error) {
    return networks.New(client).Resources.All(ctx,
        resource.WithFilter("name", "server-condition"),
        resource.WithFilter("is_vlan_transparent", false),
        resource.WithFilter("created_at", "2025-01-02T03:04:05+00:00"),
        resource.WithPaginated(false),
    )
}
```

The same options work through cached
`connection.Network(ctx).API.Networks.Resources` and the handwritten
`connection.Network(ctx).Networks` collection. Both use the existing shared
provider and HTTP client. Configure service fields before concurrent calls;
the provider's current token stays live across pages. The manual collection
retains its singular `network` error kind and its existing exact `ERROR`
failed-state wait policy.

## Attribute classification

The twenty-three canonical query attributes have thirty-five accepted
spellings. Wire aliases are accepted by semantic options as well:

| Canonical query attribute | Wire key |
| --- | --- |
| `description` | `description` |
| `fields` | `fields` |
| `id` | `id` |
| `ipv4_address_scope_id` | `ipv4_address_scope` |
| `ipv6_address_scope_id` | `ipv6_address_scope` |
| `is_admin_state_up` | `admin_state_up` |
| `is_port_security_enabled` | `port_security_enabled` |
| `is_router_external` | `router:external` |
| `is_shared` | `shared` |
| `limit` | `limit` |
| `marker` | `marker` |
| `name` | `name` |
| `project_id` | `project_id` |
| `provider_network_type` | `provider:network_type` |
| `provider_physical_network` | `provider:physical_network` |
| `provider_segmentation_id` | `provider:segmentation_id` |
| `sort_dir` | `sort_dir` |
| `sort_key` | `sort_key` |
| `status` | `status` |
| `tags` | `tags` |
| `any_tags` | `tags-any` |
| `not_tags` | `not-tags` |
| `not_any_tags` | `not-tags-any` |

These conditions are sent to the server. Semantic `name`, `status` and `id`
do not activate local predicates. Existing `resource.WithName` retains its
literal server hint and exact local name comparison. Existing
`resource.WithStatus` sends its `status` hint and also compares native Status
locally without regard to letter case. Raw `WithQuery("status", ...)` sends
wire data without enabling that local status comparison. Neither list lane
drops the status query.
Once `WithStatus` enables the predicate, the final last-wins status query value
also becomes its case-insensitive local target, including a later raw
`WithQuery("status", ...)` option.

The fourteen local attributes are:

| Semantic attribute | Raw Body field | Response policy |
| --- | --- | --- |
| `availability_zone_hints` | `availability_zone_hints` | Original JSON |
| `availability_zones` | `availability_zones` | Original JSON |
| `created_at` | `created_at` | Original JSON |
| `dns_domain` | `dns_domain` | Original JSON |
| `is_default` | `is_default` | Boolean truthiness |
| `is_vlan_qinq` | `vlan_qinq` | Boolean truthiness |
| `is_vlan_transparent` | `vlan_transparent` | Boolean truthiness |
| `mtu` | `mtu` | Exact integer |
| `pvlan` | `pvlan` | Boolean truthiness |
| `qos_policy_id` | `qos_policy_id` | Original JSON |
| `revision_number` | `revision_number` | Exact integer |
| `segments` | `segments` | Original JSON |
| `subnet_ids` | `subnets` | Original JSON |
| `updated_at` | `updated_at` | Original JSON |

Explicit `WithBodyFilter`/`WithBodyFilters` accept the fourteen raw names and
three attribute aliases: `subnet_ids`, `is_vlan_qinq` and
`is_vlan_transparent`. Individual options for one canonical field are
last-wins; an explicit bulk map containing both a raw name and its alias is
invalid, even if values agree. Semantic options accept the Python attribute
names above. Raw `subnets`, `vlan_qinq` and `vlan_transparent` are unknown
semantic names and are discarded; use `WithQuery` to send those literal wire
keys. `tenant_id` is also unknown to this semantic declaration. It is not
converted into `project_id` and does not acquire a local selector.

## Response values and options

Missing and null local values compare as null. In particular, the four local
bool descriptors have no false default. Response-only truthiness keeps
booleans unchanged, maps numeric zero and empty strings/arrays/objects to
false, and maps nonzero numbers and nonempty values to true. A response string
`"false"` is true. Numbers use exact decimal zero/nonzero semantics without
float64 conversion or exponent expansion. Thus an extremely small nonzero
exponent remains true in Go, while Python's JSON float parsing may underflow
it to zero. Caller filter values are unchanged: Go distinguishes booleans from
numbers and strings from numbers. Python boolean/number equality is an
intentional difference.

`mtu` and `revision_number` normalize response numbers and decimal integer
strings to exact signed integers. Integral decimal/exponent numbers are
accepted; bool, fractional values, objects and arrays fail when consumed by a
selected integer filter. Python instead permits bool-as-int, truncates floats
and converts digit-only strings, with some other values becoming zero. Go's
bounded exact-integral policy is deliberate. Native `revision_number` must
first pass its `int` decoder, so a response string fails before normalization.
`mtu` is absent from the native model and can reach the raw normalizer.

Other fields preserve original JSON values and timestamp spelling. A `Z`
timestamp and an equivalent `+00:00` timestamp are distinct local values.
The native timestamp decoder still requires compatible old no-zone or standard
RFC3339 formats when both timestamps are present; mixed formats may fail.
Array order, length and complete elements matter. Objects inside arrays
compare exactly; standalone object filters recursively match a subset of a
nonempty object. An empty actual object does not match an object filter.
Python list descriptors wrap non-list values into arrays; this Go lane retains
raw JSON and performs no such wrapping.

The explicit `subnets/subnet_ids` selector previously marshaled the native
`[]string` field. It now matches original JSON: missing/null remain equivalent,
empty arrays remain distinct, and null elements remain null. Returned native
`Subnets` and `AvailabilityZoneHints` still project a null element to `""`.
This changes selected-value fidelity without changing native typed List or
the returned model.

Options snapshot their inputs and can be reused concurrently. Semantic bulk
options replace only their own namespace; nil/empty maps clear that namespace
without clearing explicit Body filters or raw query values. Only final
semantic values are validated, so a later replacement can clear an earlier
invalid semantic value. In a semantic bulk map, a canonical query attribute
wins over its wire alias by key presence, including nil, false and empty.
Individual query options targeting one wire key are last-wins. Known query
scalars become URL values (bool is lowercase, numbers retain lexical spelling);
scalar arrays create repeated values, omitting null elements. Nil/empty arrays
omit the wire value but preserve key presence for conflict checks. Objects and
nested arrays are invalid query values.

Semantic query values conflict with raw query/page-size values for the same
wire key and with the name hint produced by `WithName`. Semantic `status`
therefore conflicts with `WithStatus`'s query. Semantic and explicit Body
conditions for one canonical field also conflict. Body and raw query
namespaces are independent. Unknown semantic names are discarded without
validating their values. Source controls `allow_unknown_params`, `base_path`,
`headers`, `jmespath_filters`, `max_items`, `microversion`, `paginated`,
`resource_type` and `session` require dedicated supported options; semantic
values for these controls fail lazily before HTTP.

## Native and Python boundaries

Native extraction validates the complete current page before raw cap or local
comparison. Incompatible nonnull known strings, booleans, integers, arrays or
timestamps remain native errors, including malformed rows after the cap.
Unknown extension fields can be compared without appearing on the native
model. A null row consumed by a Body-filter iterator is invalid; query-only
null rows retain the native zero Network model. Empty object rows are valid.
The raw cap and consumer break avoid unconsumed selected-field normalization.
They do not bypass native whole-page decoding.

`WithMaxItems` counts raw rows before local conditions, as Python does, but
does not inject Python's limit hint. `WithPaginated(false)` and consumer break
stop before reading continuation. The native pager retains 200/204/300,
`networks_links` with its last `rel=next`, JSON-empty-204 EOF, native foreign
continuation following and existing terminal cycle/context/error behavior.
Top-level `next`, `links.next` and HTTP Link headers do not acquire new native
pagination support. `All` returns nil with a terminal error rather than a
partially collected slice.

Native typed `API.List`, `FindIdentity`, explicit Ref lookup and CRUD retain
their existing contracts. Complete Python eager descriptor/cache/unknown-Body
and dirty lifecycle, scalar-list coercion, per-call session/header/microversion/
base-path controls, JMESPath and inherited continuation behavior remain
separate. This addition closes the declared attribute classification and
selected raw-value gaps; it does not claim full Resource parity.

Sources: pinned
[Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py),
[NetworkResource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py),
[Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155),
and [field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86).
See the [common list options](../../../../docs/listing.md),
[seven API HTTP groups](../../../../api/network_list_filters_test.go) and
[root Connection evidence](../../../../connection_network_filters_test.go).
