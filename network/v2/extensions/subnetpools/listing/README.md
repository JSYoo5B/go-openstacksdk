# SubnetPool list filters

`SubnetPools.Resources.List/All` classify `resource.WithFilter` and
`resource.WithFilters` using the pinned OpenStackSDK SubnetPool declarations.
The SDK sends declared query attributes to Neutron and compares the ten local
Body attributes against the original JSON rows. It still returns native
`*subnetpools.SubnetPool` values. Builders and application predicates are not
required.

Python:

```python
pools = list(conn.network.subnet_pools(
    is_shared=False,
    fields=["id", "name", "prefixes", "default_prefixlen", "min_prefixlen", "max_prefixlen"],
    prefixes=["192.0.2.1/24"],
    default_prefix_length=24,
    max_items=20,
))
```

Go:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func listPools(ctx context.Context, client *gophercloud.ServiceClient) ([]*subnetpools.SubnetPool, error) {
    return subnetpools.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "is_shared": false,
            "fields": []string{"id", "name", "prefixes", "default_prefixlen", "min_prefixlen", "max_prefixlen"},
            "prefixes": []string{"192.0.2.1/24"},
            "default_prefix_length": 24,
        }),
        resource.WithMaxItems(20),
    )
}
```

The native decoder requires all three prefix-length fields even when a caller
uses a `fields` projection. Include them, as in this example, and include every
field needed by a local condition. `prefixes` compares the complete ordered
array; neither client-side comparison merges adjacent prefixes nor normalizes
CIDR strings.

The second example uses literal `tenant_id`, the original timestamp spelling,
and a first-page policy. `project_id` is an independent server query; it does
not supply a missing local `tenant_id`.

Python:

```python
pools = list(conn.network.subnet_pools(
    tenant_id="legacy-tenant",
    created_at="2025-01-02T03:04:05+00:00",
    minimum_prefix_length=16,
    paginated=False,
))
```

Go:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func listTenantPools(ctx context.Context, client *gophercloud.ServiceClient) ([]*subnetpools.SubnetPool, error) {
    return subnetpools.New(client).Resources.All(ctx,
        resource.WithFilter("tenant_id", "legacy-tenant"),
        resource.WithFilter("created_at", "2025-01-02T03:04:05+00:00"),
        resource.WithBodyFilter("minimum_prefix_length", 16),
        resource.WithPaginated(false),
    )
}
```

The same options work through the cached
`connection.Network(ctx).API.SubnetPools.Resources` collection. Configure the
service client before concurrent calls; the provider, current token and HTTP
client remain shared.

## Attribute classification

The sixteen canonical query attributes have twenty accepted spellings:

| Canonical query attribute | Wire key |
| --- | --- |
| `address_scope_id` | `address_scope_id` |
| `any_tags` | `tags-any` |
| `description` | `description` |
| `fields` | `fields` |
| `ip_version` | `ip_version` |
| `is_default` | `is_default` |
| `is_shared` | `shared` |
| `limit` | `limit` |
| `marker` | `marker` |
| `name` | `name` |
| `not_any_tags` | `not-tags-any` |
| `not_tags` | `not-tags` |
| `project_id` | `project_id` |
| `sort_dir` | `sort_dir` |
| `sort_key` | `sort_key` |
| `tags` | `tags` |

These attributes are server conditions. Semantic `name` does not activate a
local name predicate. Existing `resource.WithName` supplies its literal server
hint and exact local predicate. Semantic `id` belongs to the local table below,
although `resource.WithQuery("id", ...)` remains an independent wire extension.

The ten local attributes and their raw response fields are:

| Semantic attribute | Raw Body field | Response policy |
| --- | --- | --- |
| `created_at` | `created_at` | Original JSON |
| `default_prefix_length` | `default_prefixlen` | Exact integer |
| `default_quota` | `default_quota` | Exact integer |
| `id` | `id` | Original JSON |
| `maximum_prefix_length` | `max_prefixlen` | Exact integer |
| `minimum_prefix_length` | `min_prefixlen` | Exact integer |
| `prefixes` | `prefixes` | Original JSON |
| `revision_number` | `revision_number` | Exact integer |
| `tenant_id` | `tenant_id` | Original JSON |
| `updated_at` | `updated_at` | Original JSON |

`WithBodyFilter`/`WithBodyFilters` support the ten raw field names and the three
prefix-length attribute aliases. Individual options for the same canonical
Body field are last-wins; one explicit bulk map containing an alias and its raw
field is invalid. Semantic `WithFilter` accepts the Python attribute names:
`default_prefixlen`, `min_prefixlen` and `max_prefixlen` are unknown semantic
attributes and are discarded. Raw `WithQuery` retains those names on the wire.

The `id` selector uses the literal lowercase response field. Missing/null
matches a null filter; an empty string is distinct. SubnetPool declares no
alternate-ID property, and this selector does not infer an ID from `name`.
`tenant_id` is also literal and does not fall back to `project_id`.

After successful native decoding, raw `prefixes` distinguishes missing/null,
empty arrays and null array elements. Native returned `Prefixes` remains a
`[]string`: missing/null produces a nil slice and a null element projects to
`""`. This improves the prior explicit Body selector's selected-value fidelity
without changing native typed List or the returned model.

Timestamp comparisons retain original strings. A `Z` timestamp and an
equivalent `+00:00` timestamp differ as local values, even when their native
`time.Time` values represent the same instant. Both older no-`Z` strings and
RFC3339 strings can decode natively, but the two present timestamps must fit
one compatible decoder pass. A row mixing an old no-`Z` timestamp with an
RFC3339 timestamp fails native decoding.

## Integer and native decoder boundaries

The five integer response descriptors use the SDK's bounded-memory exact
integer normalizer. Integral decimal/exponent JSON numbers compare by exact
value. Decimal integer strings, including signs, normalize to numbers only
where the native decoder first accepts them. Caller filter strings remain
strings: `"24"` does not match a normalized numeric response. Missing/null
integer fields compare as null when native decoding permits their omission.

This is a deliberate Go policy. Python's `fields._convert_type` uses digit-only
string conversion, converts numeric fractions with `int` truncation, and can
retain booleans through Python's `int` subtype/equality rules. Go rejects a
selected nonintegral response rather than truncating it, distinguishes booleans
from numbers, and accepts signed decimal strings in reachable prefix fields.
Python can turn signed or otherwise non-digit strings into zero. These are not
claims of complete descriptor-coercion equivalence.

Native `SubnetPool.UnmarshalJSON` remains authoritative before any raw
selection. `default_prefixlen`, `min_prefixlen` and `max_prefixlen` must all be
present and must be strings accepted by `strconv.Atoi` or JSON numbers decoded
through float64. Missing/null/bool/object prefix values fail on the whole page.
A numeric fractional prefix can decode natively through truncation when that
field is unselected; selecting it invokes the exact-integer error policy.
Native `default_quota` and `revision_number` are `int` fields, so string,
fractional and boolean responses fail before raw normalization. Native integer
width, float64 range and prefix projection limitations remain; raw numeric
equality does not promise a precise returned native integer.

Other known fields keep their native schema: incompatible non-null strings,
bools, integer fields, `prefixes`/`tags` arrays and timestamps still fail.
Missing/null optional string fields keep native zero strings. Native decoding
of every current-page row precedes the local cap, including malformed rows
beyond that cap. Null rows and empty objects also fail because the custom
decoder requires the three prefix lengths; there is no successful zero-model
null-row case here.

## Options, paging and scope

Option construction snapshots JSON, maps, slices and pointer values; each
iteration owns its applied values, and an option can be reused concurrently.
Individual semantic options for one target are last-wins. A semantic bulk map
prefers the canonical query key when both spellings occur, including null,
false and empty values. Bulk replacement/clear affects only semantic options,
preserving explicit Body conditions and raw query. Only final selected
semantic values are validated: a valid override or clear removes a superseded
captured error. Unknown semantic attributes are discarded even when their
values are not JSON-serializable.

Query values accept strings, booleans, numbers and arrays of scalars; scalar
arrays produce repeated values, null members are omitted, and null/empty
arrays omit URL values while preserving option key presence. Object or nested
array query values fail lazily before HTTP. Semantic query/raw query,
semantic limit/page-size, semantic name/`WithName`, and semantic Body/explicit
Body collisions fail deterministically before HTTP even when values agree.
Raw query and semantic Body conditions with the same textual field name are
independent.

`max_items` and `paginated` require `WithMaxItems` and `WithPaginated`. The other
reserved control names are `allow_unknown_params`, `base_path`, `headers`,
`jmespath_filters`, `microversion`, `resource_type` and `session`; they do not
become field filters. Raw status remains a wire extension, while typed
`WithStatus` is unsupported because native SubnetPool has no status field.

Raw caps count rows before local Body/name matching and do not refill with
matches or infer a wire limit. Python also counts raw rows; its automatic
max-items limit hint and continuation observation differ. First-page and
consumer-break stopping retain the common native stream behavior. Selected
integer normalization happens only on consumed rows after the cap, before
local name comparison. Native whole-page decoder errors still precede it.
`All` returns a nil result on a terminal late-page error.

The native pager retains accepted codes 200/204/300 and its own
`subnetpools_links` continuation, where the last `rel="next"` link wins. Top-level
`next`, `links.next` and HTTP Link are not native continuation sources. Native
advertised foreign URLs are followed; no new same-origin guard or synthetic
marker fallback is added by these selectors. A JSON-labelled empty 204 can
still produce EOF before the native empty-page shortcut. This is separate
from any stricter lookup/transport policy elsewhere in the SDK.

These options apply to ordinary `Resources.List/All`. Native typed `List`,
`FindIdentity`, explicit Ref APIs and `AddPrefixes`/`RemovePrefixes` retain
their existing contracts. The raw comparison lane preserves selected original
JSON fields without adding unknown response fields or HTTP metadata to the
native model. Python's scalar-to-list conversion, eager constructor coercion,
seeded Resource/cache/dirty lifecycle, generic controls, deprecated JMESPath,
and complete limit/marker/Link continuation are separate inherited contracts.
SubnetPool inherits `Resource` and `TagMixinNetwork`, and declares no resource
maximum microversion; no NetworkResource inheritance or version gate is
inferred.

The source is pinned to
[OpenStackSDK SubnetPool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py),
[the direct proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7667),
and [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/subnetpools/results.go).
See [common listing options](../../../../../docs/listing.md),
[seven API HTTP groups](../../../../../api/subnet_pool_list_filters_test.go),
and [the Connection integration](../../../../../connection_subnet_pool_filters_test.go).
The HTTP proofs use mock servers, not a live Neutron cloud.
