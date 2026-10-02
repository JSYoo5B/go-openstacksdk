# Router list filters

`Routers.Resources.List/All` classify `resource.WithFilter` and
`resource.WithFilters` using the pinned OpenStackSDK Router declarations.
The SDK sends eighteen declared query attributes to Neutron and compares ten
local Body attributes against original JSON rows after native page decoding.
It returns the existing native `*routers.Router` model. Applications use
library-owned options without request builders or callback predicates.

Python:

```python
routers = list(conn.network.routers(
    project_id="project-one",
    fields=["id", "name", "revision", "revision_number", "enable_ndp_proxy"],
    enable_ndp_proxy=True,
    revision_number=7,
    max_items=20,
))
```

Go:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/network/v2/extensions/layer3/routers"
    "gophercloudsdk/resource"
)

func listRouters(ctx context.Context, client *gophercloud.ServiceClient) ([]*routers.Router, error) {
    return routers.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "project_id": "project-one",
            "fields": []string{"id", "name", "revision", "revision_number", "enable_ndp_proxy"},
            "enable_ndp_proxy": true,
            "revision_number": 7,
        }),
        resource.WithMaxItems(20),
    )
}
```

Include each raw field needed by a local condition when using `fields`.
The Python `revision_number` descriptor compares raw `revision`, while the
native Go model's `RevisionNumber` reads `revision_number`. These fields are
independent: matching revision 7 does not imply that returned `RevisionNumber`
is 7. Local values absent from the native model remain available for comparison
without adding model fields.

Python:

```python
routers = list(conn.network.routers(
    name="server-condition",
    external_gateway_info={"network_id": "external-network"},
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
    "gophercloudsdk/network/v2/extensions/layer3/routers"
    "gophercloudsdk/resource"
)

func firstPageRouters(ctx context.Context, client *gophercloud.ServiceClient) ([]*routers.Router, error) {
    return routers.New(client).Resources.All(ctx,
        resource.WithFilter("name", "server-condition"),
        resource.WithFilter("external_gateway_info", map[string]any{
            "network_id": "external-network",
        }),
        resource.WithFilter("created_at", "2025-01-02T03:04:05+00:00"),
        resource.WithPaginated(false),
    )
}
```

The same options work through cached
`connection.Network(ctx).API.Routers.Resources`. This facade uses the existing
shared provider and HTTP client. Configure service fields before concurrent
calls; the provider's current token stays live across pages. There is no
separate handwritten Router collection in the Network service.

## Attribute classification

The eighteen canonical query attributes have twenty-four accepted spellings.
Wire aliases are accepted by semantic options as well:

| Canonical query attribute | Wire key |
| --- | --- |
| `description` | `description` |
| `fields` | `fields` |
| `flavor_id` | `flavor_id` |
| `id` | `id` |
| `is_admin_state_up` | `admin_state_up` |
| `is_distributed` | `distributed` |
| `is_ha` | `ha` |
| `limit` | `limit` |
| `marker` | `marker` |
| `name` | `name` |
| `project_id` | `project_id` |
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
locally without regard to letter case. Once `WithStatus` enables that
predicate, the final last-wins status query value is also its local target,
including a later raw `WithQuery("status", ...)` option. Raw status query alone
does not enable the local predicate. Both list lanes preserve the status query.

The ten local attributes are:

| Semantic attribute | Raw Body field | Response policy |
| --- | --- | --- |
| `availability_zone_hints` | `availability_zone_hints` | Original JSON |
| `availability_zones` | `availability_zones` | Original JSON |
| `created_at` | `created_at` | Original JSON |
| `enable_ndp_proxy` | `enable_ndp_proxy` | Boolean truthiness |
| `evpn_vni` | `evpn_vni` | Exact integer |
| `external_gateway_info` | `external_gateway_info` | Original JSON |
| `revision_number` | `revision` | Exact integer |
| `routes` | `routes` | Original JSON |
| `tenant_id` | `tenant_id` | Original JSON |
| `updated_at` | `updated_at` | Original JSON |

Explicit `WithBodyFilter`/`WithBodyFilters` accept the ten raw names and the
`revision_number` alias for `revision`. Individual options for one canonical
field are last-wins; an explicit bulk map containing both `revision` and
`revision_number` is invalid even if values agree. Semantic options accept the
Python attribute names above. Raw `revision` is an unknown semantic name and
is discarded; use `WithQuery` to send that literal wire key. `tenant_id` is
local and independent of query `project_id`; it does not fall back to the
native ProjectID or infer a tenant query. Query flags such as `is_ha` do not
acquire Body selectors.

## Response values and options

Missing and null local values compare as null. `enable_ndp_proxy` has no false
default. Response-only truthiness keeps booleans unchanged, maps numeric zero
and empty strings/arrays/objects to false, and maps nonzero numbers and
nonempty values to true. A response string `"false"` is true. Numbers use
exact decimal zero/nonzero semantics without float64 conversion or exponent
expansion. An extremely small nonzero exponent remains true in Go, while
Python's JSON float parsing may underflow it to zero. Caller filter values
are unchanged: Go distinguishes booleans from numbers and strings from
numbers; Python boolean/number equality is intentionally different.

`evpn_vni` and raw `revision` normalize response numbers and decimal integer
strings to exact signed integers. Integral decimal/exponent numbers are
accepted; bool, fractional values, objects and arrays fail when consumed by a
selected integer filter. Python instead permits bool-as-int, truncates floats
and converts digit-only strings, with some other values becoming zero. Go's
bounded exact-integral policy is deliberate. Neither raw field is decoded by
the native Router model; native `revision_number` must independently pass its
`int` decoder before any local comparison or cap.

Other fields preserve original JSON values and timestamp spelling. A `Z`
timestamp and an equivalent `+00:00` timestamp are distinct local values.
The native timestamp decoder still requires compatible old no-zone or standard
RFC3339 formats when both timestamps are present; mixed formats may fail.
Array order, length and complete elements matter. Objects inside arrays compare
exactly; standalone object filters recursively match a subset of a nonempty
object. An empty actual object does not match an object filter. Python list
descriptors wrap non-list values into arrays and its dict descriptor converts
nonnull nonobjects to empty dictionaries; this Go lane retains raw JSON and
performs neither coercion.

Raw Gateway/Route objects retain unknown nested keys, exact numbers and null
members for filtering. Returned native `GatewayInfo`, `ExternalFixedIP` and
`Route` still expose only their declared fields: null array elements become
zero structs, a null `enable_snat` becomes a nil pointer, and null availability
hint elements become `""`. Filtering a complete array containing null differs
from filtering the native projection of that array.

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
wire key and with hints produced by `WithName`/`WithStatus`. Semantic and
explicit Body conditions for one canonical field also conflict. Body and raw
query namespaces are independent, including a raw `revision` query combined
with a semantic `revision_number` condition. Unknown semantic names are
discarded without validating their values. Source controls
`allow_unknown_params`, `base_path`, `headers`, `jmespath_filters`, `max_items`,
`microversion`, `paginated`, `resource_type` and `session` require dedicated
supported options; semantic values for them fail lazily before HTTP.

## Native and Python boundaries

Native extraction validates the complete current page before raw cap or local
comparison. Incompatible nonnull known strings, booleans, integers, arrays,
nested Gateway/Route fields or timestamps remain native errors, including
malformed rows after the cap. Null known fields keep their native zero values.
A null row consumed by a Body-filter iterator is invalid; query-only null rows
retain the native zero Router model. Empty object rows are valid. Raw caps and
consumer break avoid unconsumed selected-field normalization, while preserving
native whole-page decoding.

`WithMaxItems` counts raw rows before local conditions, as Python does, but
does not inject Python's limit hint. `WithPaginated(false)` and consumer break
stop before reading continuation. The native pager retains 200/204/300,
`routers_links` with its last `rel=next`, JSON-empty-204 EOF, native foreign
continuation following and existing terminal cycle/context/error behavior.
Top-level `next`, `links.next` and HTTP Link headers do not acquire new native
pagination support. `All` returns nil with a terminal error rather than a
partially collected slice. No default limit or marker fallback is introduced.

Native typed `API.List`, `Get`, `FindIdentity`, explicit Ref lookup and Router
actions retain their existing contracts. Complete Python eager descriptor,
mutable Resource cache/dirty lifecycle, scalar-list/dict coercion, per-call
session/header/microversion/base-path controls, JMESPath and inherited
continuation behavior remain separate. This addition closes the declared
attribute classification and selected raw-value gaps; it does not claim full
Resource parity.

Sources: pinned
[Router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/router.py),
[NetworkResource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py),
[Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155),
[field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86),
and native
[Router results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/layer3/routers/results.go).
See the [common list options](../../../../../../docs/listing.md),
[seven API HTTP groups](../../../../../../api/router_list_filters_test.go)
and [root Connection evidence](../../../../../../connection_router_filters_test.go).
