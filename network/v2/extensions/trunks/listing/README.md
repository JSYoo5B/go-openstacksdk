# Trunk list filters

`Trunks.Resources.List/All` classify `resource.WithFilter` and
`resource.WithFilters` using the pinned OpenStackSDK Trunk declarations.
Fourteen query attributes go to Neutron; two local Body attributes compare
original JSON rows after native whole-page extraction. Results remain the
existing native `*trunks.Trunk` model. The SDK supplies request builders and
filtering without application callbacks.

Python:

```python
trunks = list(conn.network.trunks(
    project_id="project-one",
    is_admin_state_up=False,
    fields=["id", "name", "project_id", "tenant_id"],
    id="selected-trunk",
    tenant_id="tenant-one",
    max_items=20,
))
```

Go:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func listTrunks(ctx context.Context, client *gophercloud.ServiceClient) ([]*trunks.Trunk, error) {
    return trunks.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "project_id": "project-one",
            "is_admin_state_up": false,
            "fields": []string{"id", "name", "project_id", "tenant_id"},
            "id": "selected-trunk",
            "tenant_id": "tenant-one",
        }),
        resource.WithMaxItems(20),
    )
}
```

`id` and `tenant_id` are local conditions; `project_id` is a server condition.
No current project or tenant filter is inferred. Include every local field
needed for matching when requesting a `fields` projection.

Python:

```python
trunks = list(conn.network.trunks(
    name="server-condition",
    status="BUILD",
    tenant_id=None,
    fields=["id", "name", "status", "tenant_id"],
    paginated=False,
))
```

Go:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func firstPageTrunks(ctx context.Context, client *gophercloud.ServiceClient) ([]*trunks.Trunk, error) {
    return trunks.New(client).Resources.All(ctx,
        resource.WithFilter("name", "server-condition"),
        resource.WithFilter("status", "BUILD"),
        resource.WithFilter("tenant_id", nil),
        resource.WithFilter("fields", []string{"id", "name", "status", "tenant_id"}),
        resource.WithPaginated(false),
    )
}
```

These options also work through cached
`connection.Network(ctx).API.Trunks.Resources`. The API uses the existing
shared provider and HTTP client. Configure service fields before concurrent
calls; the provider's current token stays live between pages. The Network
service has no separate handwritten Trunk collection.

## Attribute classification

The fourteen canonical query attributes have eighteen accepted spellings.
Semantic options also accept the wire aliases:

| Canonical query attribute | Wire key |
| --- | --- |
| `description` | `description` |
| `fields` | `fields` |
| `is_admin_state_up` | `admin_state_up` |
| `limit` | `limit` |
| `marker` | `marker` |
| `name` | `name` |
| `port_id` | `port_id` |
| `project_id` | `project_id` |
| `status` | `status` |
| `sub_ports` | `sub_ports` |
| `tags` | `tags` |
| `any_tags` | `tags-any` |
| `not_tags` | `not-tags` |
| `not_any_tags` | `not-tags-any` |

All these conditions are server-only. Semantic `name` and `status` do not
enable local predicates. `sub_ports` stays a server query even though its
Python response descriptor has type `list` and native results contain nested
Subport structs. `is_admin_state_up` is also a query, so its Python response
Boolean descriptor introduces no local normalization.

Trunk inherits `Resource` and `TagMixin`, rather than `NetworkResource`.
Consequently native fields `created_at`, `updated_at` and `revision_number`
are not declared semantic attributes for this Python resource. Unknown
semantic names, including those fields and native-only `sort_key`/`sort_dir`,
are discarded even when their values are invalid JSON. Raw query options can
still pass literal wire extension keys to the server.

The two local attributes are:

| Semantic attribute | Raw Body field | Response policy |
| --- | --- | --- |
| `id` | `id` | Original JSON |
| `tenant_id` | `tenant_id` | Original JSON |

Explicit `WithBodyFilter`/`WithBodyFilters` accept exactly those two names.
`project_id` has a Python response alias to `tenant_id`; that alias does not
turn local tenant matching into a project query or infer a missing tenant
from native `ProjectID`. The Trunk name descriptor is not an alternate ID:
missing/null `id` never falls back to `name`.

Raw `WithQuery("id", ...)` and `WithQuery("tenant_id", ...)` remain wire
extensions independent of the local conditions with those names. Existing
`WithName` retains its literal server hint plus exact local name comparison.
`WithStatus` retains a wire status and a case-insensitive local comparison.
Once `WithStatus` enables that comparison, the final last-wins status query
value is also the local target; a later raw status option changes both. A raw
status option alone does not enable a local predicate.

## Values, ownership and controls

Missing and null local values compare as null, while explicit empty strings
remain distinct. The native model returns empty strings for missing/null
ID and TenantID, but matching uses the original fields. Arbitrary valid JSON
filter values are accepted without coercion; a Boolean, number, array or
object does not match a returned string. Go distinguishes booleans from
numbers; Python Boolean/number equality is intentionally different.
Standalone object comparison uses recursive subsets with empty actual
objects failing object filters, and arrays compare their complete ordered
elements. Native string decoding limits which response values can actually
reach these two local comparisons.

Options snapshot inputs at construction and application and can be reused
concurrently. Semantic bulk options replace their own namespace; nil/empty
maps clear only that namespace, preserving explicit Body filters and raw
queries. Only final selected semantic values are validated, so replacement
or clear can remove an earlier invalid value. Canonical query attributes
win over their wire aliases by key presence in a bulk map, including nil,
false and empty values. Individual options targeting one wire key use the
last value.

Query scalars become URL values; booleans are lowercase and JSON numbers
retain lexical spelling. Scalar arrays create repeated values, omitting
null elements. Nil/empty arrays omit the wire value but retain key presence
for collision checks. Objects and nested arrays are invalid query values.
Semantic query options conflict with raw query/page-size values for the same
wire key, and with the name/status hints enabled by `WithName`/`WithStatus`.
Semantic and explicit Body filters for one field also conflict, even when
their values agree. Body and raw query namespaces are independent.

Source controls `allow_unknown_params`, `base_path`, `headers`,
`jmespath_filters`, `max_items`, `microversion`, `paginated`, `resource_type`
and `session` require dedicated supported options. Semantic values for them
fail lazily before HTTP. `WithMaxItems` counts raw rows before local filters,
as Python does, but does not inject Python's limit hint. `WithPaginated(false)`
and consumer break stop before reading continuation.

## Native and Python boundaries

The native Trunk and Subport schemas decode the entire current page before
raw caps or local comparisons. Incompatible nonnull known strings, Boolean
fields, integers, arrays, nested Subport fields and timestamps remain native
errors, including malformed later rows beyond the cap. Trunk has no custom
timestamp decoder: its `time.Time` fields accept standard RFC3339 forms,
including `Z`/offsets, while old no-zone strings fail. Missing/null known
fields retain native zero values. Null Subport array elements become zero
structs; missing/null arrays remain nil and empty arrays remain empty.

A null row consumed by a Body-filter iterator is invalid. Query-only null
rows retain native zero Trunk models, and empty object rows are valid.
Caps and break avoid unconsumed local selector work without bypassing native
whole-page validation. Returned models remain native projections; raw rows
are used for the two local comparisons, not attached as fabricated metadata.

The existing library-owned native List builder and pager are retained.
Repeated/nil/empty raw queries, provider/client/prefix and raw status are
preserved. Native pagination accepts 200/204/300 and uses inherited top-level
`links.next`; `trunks_links`, top-level `next` and HTTP Link headers do not
acquire pagination support. Native foreign continuations are followed;
malformed links, cycles, late HTTP/decode errors and cancellation are
terminal. JSON-empty 204 produces EOF before native empty-page handling.
`All` returns nil on a terminal error rather than a partially collected
slice. No default limit, synthetic marker or new origin guard is added.

Typed native `API.List`, `Get`, `FindIdentity`, explicit Ref lookup, CRUD,
tags and subport actions retain their existing contracts. Native ListOpts
does not expose fields/limit/marker/sub_ports, but library-owned raw query
options can supply them. Python's permissive descriptors, mutable Resource
cache/dirty state, eager conversion, per-call session/header/microversion/
base-path controls, JMESPath and inherited continuation policy remain
separate. This unit closes attribute classification and selected raw-value
gaps without claiming complete Resource parity.

Sources: pinned
[Trunk](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/trunk.py),
[Proxy.trunks](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7911),
[Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155),
[TagMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py),
native [Trunk results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/trunks/results.go)
and [LinkedPageBase](https://github.com/gophercloud/gophercloud/blob/v2.15.0/pagination/linked.go).
See the [common list options](../../../../../docs/listing.md),
[seven HTTP groups](../../../../../api/trunk_list_filters_test.go) and
[two Connection groups](../../../../../connection_trunk_filters_test.go).
