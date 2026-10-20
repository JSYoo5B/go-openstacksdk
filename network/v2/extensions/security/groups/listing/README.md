# SecurityGroup list filters

`SecurityGroups.Resources.List/All` classify `resource.WithFilter` and
`resource.WithFilters` using the pinned OpenStackSDK SecurityGroup declarations.
The SDK sends seventeen declared query attributes to Neutron and compares
three local Body attributes against original JSON rows after native page
extraction. It returns the existing native `*groups.SecGroup` model.
Applications supply library-owned options without request builders or callback
predicates.

Python:

```python
groups = list(conn.network.security_groups(
    is_shared=False,
    project_id="project-one",
    fields=["id", "name", "created_at", "updated_at", "security_group_rules"],
    created_at="2025-01-02T03:04:05+00:00",
    security_group_rules=[],
    max_items=20,
))
```

Go:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func listSecurityGroups(ctx context.Context, client *gophercloud.ServiceClient) ([]*groups.SecGroup, error) {
    return groups.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "is_shared": false,
            "project_id": "project-one",
            "fields": []string{"id", "name", "created_at", "updated_at", "security_group_rules"},
            "created_at": "2025-01-02T03:04:05+00:00",
            "security_group_rules": []any{},
        }),
        resource.WithMaxItems(20),
    )
}
```

Include every raw field needed by a local condition when using `fields`.
`security_group_rules` compares the complete ordered array. It does not select
rules by membership or match a subset of each rule. Unknown nested rule fields
remain available for comparison without adding fields to the returned model.

Python:

```python
groups = list(conn.network.security_groups(
    name="server-condition",
    stateful=False,
    updated_at="2025-01-03T03:04:05",
    paginated=False,
))
```

Go:

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func firstPageSecurityGroups(ctx context.Context, client *gophercloud.ServiceClient) ([]*groups.SecGroup, error) {
    return groups.New(client).Resources.All(ctx,
        resource.WithFilter("name", "server-condition"),
        resource.WithFilter("stateful", false),
        resource.WithFilter("updated_at", "2025-01-03T03:04:05"),
        resource.WithPaginated(false),
    )
}
```

The same options work through cached
`connection.Network(ctx).API.SecurityGroups.Resources`. This facade uses the
existing shared provider and HTTP client. Configure service fields before
concurrent calls; the provider's current token stays live across pages.
The Network service has no separate handwritten SecurityGroup collection.

## Attribute classification

The seventeen canonical query attributes have twenty-one accepted spellings.
Wire aliases are accepted by semantic options as well:

| Canonical query attribute | Wire key |
| --- | --- |
| `description` | `description` |
| `fields` | `fields` |
| `id` | `id` |
| `is_shared` | `shared` |
| `limit` | `limit` |
| `marker` | `marker` |
| `name` | `name` |
| `project_id` | `project_id` |
| `revision_number` | `revision_number` |
| `sort_dir` | `sort_dir` |
| `sort_key` | `sort_key` |
| `stateful` | `stateful` |
| `tenant_id` | `tenant_id` |
| `tags` | `tags` |
| `any_tags` | `tags-any` |
| `not_tags` | `not-tags` |
| `not_any_tags` | `not-tags-any` |

These conditions are server-only. Semantic `name` and `id` do not activate
local predicates. `project_id` and `tenant_id` remain separate wire keys;
the Python project's response alias does not rewrite either query.
`stateful`, `is_shared` and inherited `revision_number` also remain queries,
so their response descriptor types do not introduce local Boolean or Integer
normalization.

Existing `resource.WithName` retains its literal server hint and exact local
name comparison. Native SecGroup has no Status field, so `resource.WithStatus`
is unsupported before HTTP. Raw `WithQuery("status", ...)` still forwards
wire data without enabling a local comparison. Semantic `status` is unknown
and is discarded, even when its value is invalid JSON.

The three local attributes are:

| Semantic attribute | Raw Body field | Response policy |
| --- | --- | --- |
| `created_at` | `created_at` | Original JSON |
| `security_group_rules` | `security_group_rules` | Original JSON |
| `updated_at` | `updated_at` | Original JSON |

Explicit `WithBodyFilter`/`WithBodyFilters` accept exactly these three names.
They do not turn queried `revision_number`, `tenant_id`, `stateful`, `shared`,
`name` or `id` into local selectors. `rules` is not a semantic alias for
`security_group_rules`; it is an unknown semantic key. Raw query options can
send literal extension keys independently of local Body conditions.

## Response values and options

Missing and null local values compare as null; empty arrays are distinct.
Raw null rule elements remain null for matching while the native returned
`Rules` slice contains zero `SecGroupRule` structs at those positions.
Missing/null rule arrays remain native nil slices, while an empty array remains
empty. Native known null fields retain their zero string, integer or bool value.

Array order, length and complete elements matter. Objects inside arrays
compare exactly, including unknown nested members; they are not object-subset
conditions. Raw numbers compare as exact decimals after successful native
extraction, so an unknown rule member containing `9007199254740993` is distinct
from `9007199254740992`. Go distinguishes booleans from numbers and strings
from numbers; Python boolean/number equality is intentionally different.
Standalone object filters use recursive subset comparison on nonempty objects,
with an empty actual object failing an object filter. The rules response must
first satisfy the native array schema, so a scalar/object response for that
field can fail before comparison. Python's list descriptor can wrap non-list
values into a list; this Go lane does not perform that coercion.

Raw timestamps compare by their original text: a `Z` timestamp and an equivalent
`+00:00` timestamp are distinct local values. Group and nested Rule each have
their own native decoder. Within each object, timestamp pairs must use
compatible old no-zone or standard RFC3339 forms; mixed formats may fail.
A no-zone Group and a standard RFC3339 Rule can coexist. Local comparison does
not bypass either decoder.

Options snapshot their inputs and can be reused concurrently. Semantic bulk
options replace only their own namespace; nil/empty maps clear that namespace
without clearing explicit Body filters or raw queries. Only final semantic
values are validated, so a later replacement can clear an earlier invalid
semantic value. In a semantic bulk map, a canonical query attribute wins over
its wire alias by key presence, including nil, false and empty. Individual
options targeting one wire key are last-wins. Query scalars become URL values
(bool is lowercase, numbers retain lexical spelling); scalar arrays create
repeated values, omitting null elements. Nil/empty arrays omit the wire value
but preserve key presence for collision checks. Objects and nested arrays are
invalid query values.

Semantic query conditions conflict with raw query/page-size values for the
same wire key and with the name hint produced by `WithName`. Semantic and
explicit Body conditions for one canonical field also conflict. Body and raw
query namespaces remain independent, including raw `security_group_rules`
queries paired with local conditions for that field. Unknown semantic names
are discarded without validating their values. Source controls
`allow_unknown_params`, `base_path`, `headers`, `jmespath_filters`, `max_items`,
`microversion`, `paginated`, `resource_type` and `session` require dedicated
supported options; semantic values for them fail lazily before HTTP.

## Native and Python boundaries

The native SecGroup decoder and nested SecGroupRule decoder validate the entire
current page before raw cap or local comparison. Incompatible nonnull known
strings, booleans, integers, arrays or timestamps remain native errors,
including malformed rule fields after the cap. Native port/revision integers
must decode successfully; exact raw filtering does not remove those limits.
Unknown rule JSON is preserved for comparison only, while returned models
remain native projections.

A null row consumed by a Body-filter iterator is invalid; query-only null rows
retain the native zero SecGroup model. Empty object rows are valid. Raw caps
and consumer break avoid unconsumed local comparison without skipping native
whole-page validation. `WithMaxItems` counts raw rows before local conditions,
as Python does, but does not inject Python's limit hint. `WithPaginated(false)`
and consumer break stop before reading continuation.

The existing SDK-owned raw-query pager is retained because native List accepts
concrete ListOpts and cannot represent all raw wire values. Both query-only
and Body lanes preserve repeated/nil/empty query ownership, fields/shared/vendor
extensions and raw status. Native pagination retains 200/204/300,
`security_groups_links` with its last `rel=next`, JSON-empty-204 EOF, native
foreign continuation following and terminal cycle/context/error behavior.
Top-level `next`, `links.next` and HTTP Link headers do not acquire new native
pagination support. `All` returns nil with a terminal error rather than a
partially collected slice. No default limit or marker fallback is introduced.

Native typed `API.List`, `Get`, `FindIdentity`, explicit Ref lookup, CRUD and
SecurityGroupRule APIs retain their existing contracts. Complete Python eager
descriptor, mutable Resource cache/dirty lifecycle, scalar-list coercion,
per-call session/header/microversion/base-path controls, JMESPath and inherited
continuation behavior remain separate. This addition closes the declared
attribute classification and selected raw-value gaps; it does not claim full
Resource parity.

Sources: pinned
[SecurityGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group.py),
[NetworkResource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py),
[Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155),
[field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86),
and native
[Group results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/security/groups/results.go)
and [Rule results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/security/rules/results.go).
See the [common list options](../../../../../../docs/listing.md),
[seven API HTTP groups](../../../../../../api/security_group_list_filters_test.go),
[four raw iterator HTTP groups](../../../../../../internal/nativefind/security_group_bodies_test.go)
and [root Connection evidence](../../../../../../connection_security_group_filters_test.go).
