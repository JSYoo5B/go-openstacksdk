# Container list filters

`containers.API.Resources.List` and `All` support semantic query and local Body
filters through `resource.WithFilter` and `WithFilters`. Results remain native
`containers.Container` models. Filtering original JSON does not add unknown
response fields to those models, fetch containers again, or follow references.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func listCertificateContainers(ctx context.Context, client *gophercloud.ServiceClient) ([]*containers.Container, error) {
    return containers.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "name": "certificate-container",
            "type": "certificate",
        }),
        resource.WithMaxItems(20))
}
```

The corresponding pinned OpenStackSDK call is:

```python
values = list(conn.key_manager.containers(
    name="certificate-container",
    type="certificate",
    max_items=20,
))
```

Here `name` and `type` are local attributes. No automatic name query is sent by
semantic `name`. `MaxItems` counts raw rows before filtering, so fewer than 20
models may match. This native pager does not insert a wire limit from that cap.

## Query and Body classification

Only `limit` and `marker` are semantic server-query attributes. All ten local
Body attributes are listed below; they are also the supported explicit
`WithBodyFilter` keys.

| Attribute | Original row value |
| --- | --- |
| `id` | Literal `id` when present; otherwise the full `container_ref`. |
| `name` | `name` |
| `container_ref` | `container_ref` |
| `container_id` | Passive final path component of `container_ref`. |
| `created_at` | `created` |
| `updated_at` | `updated` |
| `secret_refs` | `secret_refs` |
| `consumers` | `consumers` |
| `status` | `status` |
| `type` | `type` |

In particular, Python's Container has no name query mapping. The native
`ListOpts.Name`, raw `WithQuery("name", ...)`, and existing `WithName` server
hint remain separately available. `WithName` also enables its common local
name predicate. A raw name query and semantic local `name` can coexist; the
semantic filter always compares the original row value. No semantic aliases
are invented for native `offset` or raw `created`/`updated` field names.

Known query scalars encode strings, lowercase bools, and JSON-number lexical
spellings. Flat scalar arrays preserve repeated values and omit null elements.
Nil/empty-array values emit no URL value but retain key presence for collision
checks. Objects and nested arrays are invalid known query values. Unknown
semantic keys, including invalid JSON values, are discarded. Selected source
controls `max_items`, `paginated`, `base_path`, `allow_unknown_params`,
`headers`, `microversion`, `jmespath_filters`, `resource_type`, and `session`
require dedicated supported APIs and are rejected as semantic filters.

Individual options targeting one attribute use the last call. `WithFilters`
replaces only the semantic namespace; nil/empty maps clear it without clearing
raw queries or explicit Body filters. Only final selected semantic values are
validated, so a later replacement/clear can discard an earlier invalid value.
Existing explicit Body-option errors remain sticky. Inputs are immutable JSON
snapshots at construction and application, including nested maps and slices,
and owned options can be reused concurrently with a configured client.

Semantic queries conflict with raw queries or `WithPageSize` targeting the same
wire key, even for equal values. A semantic and explicit Body filter for the
same attribute also conflict. A local Body filter and raw query with the same
text remain independent.

## Original JSON comparisons and identities

Missing and null attributes match a nil filter; empty arrays remain distinct.
Arrays compare exact order, length, and nested values, including unknown object
keys and null elements. Decimal numbers compare exactly without float64
projection; strings are not coerced to numbers. Objects use recursive subset
matching: an empty actual object fails an object filter, while an empty filter
matches a nonempty actual object. Go deliberately distinguishes bools from
numbers, unlike Python's `True == 1`; an object filter against a nonobject is
a nonmatch rather than a Python attribute-access error.

`created_at` and `updated_at` compare the original response strings, rather than
the native model's parsed `time.Time` values. Literal `id` key presence always
wins, including null, empty, numeric, or object values. Otherwise `id` means
the full raw `container_ref`, not its final component. These values never
select a request URL or synthesize an ID in the native result.

`container_id` is a separate formatter. Missing/null references bypass it;
other values require a scheme, authority, and path, then return the final
original path component. It preserves literal Unicode/space and existing
percent escapes, excludes query/fragment, and returns an empty string for a
trailing slash. It does not validate UUIDs, unescape/clean paths, or follow the
reference. Only a selected `container_id` invokes formatting, before common
local name/status predicates. Go's URL parser rejects some inputs accepted by
Python `urlsplit`, such as nonnumeric authority ports; full parser parity is
not claimed.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func listExactContainerReferences(ctx context.Context, client *gophercloud.ServiceClient) ([]*containers.Container, error) {
    return containers.New(client).Resources.All(ctx,
        resource.WithFilter("secret_refs", []any{
            map[string]any{
                "name": "certificate",
                "secret_ref": "https://barbican.example/v1/secrets/secret-id",
            },
        }),
        resource.WithBodyFilter("status", "ACTIVE"),
        resource.WithPaginated(false))
}
```

The corresponding Python attributes and first-page control are:

```python
values = list(conn.key_manager.containers(
    secret_refs=[{
        "name": "certificate",
        "secret_ref": "https://barbican.example/v1/secrets/secret-id",
    }],
    status="ACTIVE",
    paginated=False,
))
```

This array filter requires the full original ordered value, including any
additional response keys. It does not perform object-subset matching inside
an array or follow its secret references.

## Native decoder, controls, and pagination

Native whole-page extraction runs first. Its nine typed fields and nested
`SecretRef`/`ConsumerRef` string fields remain strict; malformed arrays,
incompatible known fields, or native timestamp failures anywhere in the current
page are terminal even beyond a raw cap. Python's `type=list` descriptor may
wrap a scalar into a list, but this native array decoder does not. Raw filters
retain original unknown keys/numbers and null elements only after that decoder
succeeds. A null row consumed by a Body-filter iterator is invalid; an empty
object remains valid with missing attributes. Cap or break can skip a later
selected null row/formatter,
after native whole-page decoding succeeds.

Body matching precedes the common local name/status predicates. `WithStatus`
retains local case-insensitive matching; semantic Body `status` compares exact
JSON. Both ordinary and raw-filter Container adapters retain removal of raw
`status` query values. Raw `WithQuery("status", ...)` alone is neither a local
predicate nor a server filter. Other raw values, including repeated query
values, reach the native builder unchanged. Native typed `API.List` options
keep their own wire behavior.

The local raw cap, first-page `WithPaginated(false)`, and consumer break stop
before reading a continuation. `All` returns nil with a terminal error instead
of partial models. HTTP/decode/context causes and cycles remain observable;
duplicate matching rows remain ordinary list rows.

The native pager accepts HTTP 200/204/300. A successfully parsed 204 page is
empty; an empty 204 advertised as `application/json` fails with native JSON EOF
before the emptiness check. It follows the top-level `next` URL, including
foreign advertised URLs, and does not synthesize marker/offset fallback or add
the guarded pagination policy used by separate manual Secret Finder APIs.
This unit leaves native `Get`, `List`, explicit-Ref lookup, and consumer/CRUD
operations unchanged. It does not add heuristic `FindIdentity` to Containers.

## OpenStackSDK comparison

Pinned source is OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`:
[`containers`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L129-L140),
[`Container` descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/container.py#L17-L48),
[`Resource.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2210-L2358),
and [`HREFToUUID`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_format.py#L18-L29).

The descriptor classification and original-value comparisons are implemented
within the Go policies above. Native strict decoding, bool/number comparison,
selected-only formatting, and URL-parser boundaries differ from Python's
mutable Resource/property behavior. Python construction/`to_dict` may invoke
formatters that Go did not select. Generic inherited cache, JMESPath,
base-path/session/header controls, Python continuation/marker behavior, and
unknown-parameter controls are not claimed as complete parity. Python may
insert a wire limit from `max_items`; this native Go pager uses only a local cap.

The seven
[`TestKeyManagerContainerListFilters` HTTP groups](../../../../api/keymanager_container_list_filters_test.go)
verify both leaf and cached Connection paths, all ten properties, passive IDs,
exact arrays/numbers, ownership/preflight, native decoding/statuses, controls,
pagination, and unchanged native operations. Two separate
[Connection groups](../../../../connection_keymanager_container_filters_test.go)
provide high-level cached service integration evidence. These tests use local
servers and custom transports, not a live cloud.
