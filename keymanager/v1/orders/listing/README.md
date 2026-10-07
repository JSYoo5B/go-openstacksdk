# Order list filters

`orders.API.Resources.List` and `All` accept semantic query and local Body
filters through `resource.WithFilter` and `WithFilters`. They return native
`orders.Order` models. Original JSON is used for comparison without adding
unknown fields to the returned model, fetching associated secrets, or following
response references.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/orders"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func listAESOrders(ctx context.Context, client *gophercloud.ServiceClient) ([]*orders.Order, error) {
    return orders.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "type": "key",
            "meta": map[string]any{"algorithm": "aes", "bit_length": 256},
        }),
        resource.WithMaxItems(20))
}
```

The corresponding pinned OpenStackSDK call is:

```python
values = list(conn.key_manager.orders(
    type="key",
    meta={"algorithm": "aes", "bit_length": 256},
    max_items=20,
))
```

These attributes match locally; the metadata filter is a recursive object
subset. The Go cap counts raw rows before filtering, so fewer than 20 orders
may match. This native pager does not insert a server limit from that cap.

## Query and Body attributes

Only `limit` and `marker` are semantic server queries. All fourteen Body
attributes below are local; they are also the supported explicit
`WithBodyFilter` keys.

| Attribute | Original row value |
| --- | --- |
| `id` | Literal `id` when present; otherwise the full raw `order_ref`. |
| `name` | Exact top-level `name`; never `meta.name`. |
| `created_at` | `created` |
| `updated_at` | `updated` |
| `creator_id` | `creator_id` |
| `meta` | `meta` |
| `order_ref` | `order_ref` |
| `order_id` | Passive final component of `order_ref`. |
| `secret_ref` | `secret_ref` |
| `secret_id` | Passive final component of `secret_ref`. |
| `status` | `status` |
| `sub_status` | `sub_status` |
| `sub_status_message` | `sub_status_message` |
| `type` | `type` |

Semantic `name` is the inherited Python Body attribute, so it sends no name
query. The native model has no top-level `Name`; its `Meta.Name` is a separate
secret attribute. A raw `WithQuery("name", ...)` remains a wire extension and
can coexist with local semantic `name`. Existing explicit `resource.Name`
Find/Remove/waits and `WithName` remain unsupported before HTTP. Name support
is not synthesized from `Meta.Name`, and heuristic `FindIdentity` is not added.

Known query values encode strings, lowercase bools, and JSON-number lexical
spellings. Flat scalar arrays retain repeated values and omit null elements.
Nil/empty-array values omit URL values while retaining key presence for
collision checks. Known query objects/nested arrays are invalid. Unknown
semantic keys, including values that cannot be JSON-encoded, are discarded.
Native `offset` remains a separate typed/raw wire option; it is not a semantic
Python query attribute. Raw `created`/`updated` are not aliases for local
`created_at`/`updated_at`.

Selected source controls `max_items`, `paginated`, `base_path`,
`allow_unknown_params`, `headers`, `microversion`, `jmespath_filters`,
`resource_type`, and `session` require dedicated supported APIs and are rejected
as semantic filters. Semantic query keys conflict with raw query keys or
`WithPageSize` targeting the same wire key, even for equal values. A semantic
and explicit Body filter for the same attribute also conflict. Body filters
and raw queries with the same text remain independent.

Individual options use the last value for an attribute. `WithFilters` replaces
only the semantic namespace; nil/empty maps clear it while leaving raw queries
and explicit Body filters intact. Only final selected semantic values are
validated, so replacing or clearing an invalid value removes its captured
error. Existing explicit Body-option errors remain sticky. Inputs are owned
JSON snapshots at construction and application, including nested maps/slices,
and can be reused concurrently after configuring the client.

## Exact raw values and passive identity

Missing/null values match a nil filter. Empty arrays are distinct from null;
arrays compare exact order, length, and nested object keys/values. Object
filters use recursive subset comparison: an empty actual object fails an
object filter, while an empty filter matches a nonempty actual object. Decimal
numbers compare exactly without float64 projection, including unknown metadata
numbers beyond floating-point precision. Strings are not coerced to numbers.
Go deliberately distinguishes bools from numbers, unlike Python's `True == 1`,
and treats an object filter against a nonobject as a nonmatch rather than a
Python attribute-access error. Timestamp filters compare original strings,
rather than the model's parsed `time.Time` values.

Literal `id` presence wins even for null, empty, numeric, or object values;
otherwise its alternate is the **full** original `order_ref`. It is not
`secret_ref` or a UUID/path suffix. `order_id` and `secret_id` format different
references. Missing/null references bypass formatting; selected nonnull
references require a scheme, authority, and path, then return the original
final path component. Percent escapes and literal Unicode/space are retained;
query/fragment are excluded, and a trailing slash produces an empty string.
No UUID validation, unescaping, path cleaning, or URL following occurs.

Only selected formatter attributes are evaluated, before common local status
matching. Go's URL parser has stricter boundaries than Python `urlsplit`, such
as rejecting nonnumeric authority ports. Complete parser parity is not
claimed. Neither formatted IDs nor passive raw identities change the explicit
ID routes used by Find, Remove, Wait, or WaitDeleted.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/orders"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func listNamedOrderRows(ctx context.Context, client *gophercloud.ServiceClient) ([]*orders.Order, error) {
    return orders.New(client).Resources.All(ctx,
        resource.WithFilter("name", "top-level-order-name"),
        resource.WithBodyFilter("status", "ACTIVE"),
        resource.WithPaginated(false))
}
```

The corresponding Python row attributes and first-page control are:

```python
values = list(conn.key_manager.orders(
    name="top-level-order-name",
    status="ACTIVE",
    paginated=False,
))
```

The `name` in these calls is the top-level response field. It does not match
an order that has only `meta.name` set, and is not exposed as a new native model
field.

## Native decoder and pagination boundaries

The native whole-page decoder runs before raw matching. Its thirteen Order
fields, six Meta fields, strict integer `Meta.BitLength`, and both custom
timestamp decoders remain authoritative. Malformed known fields anywhere in
the current page are terminal even beyond a local cap. Python's dict property
conversion can accept nonobject values more broadly; the native Meta decoder
rejects them first. Raw comparison preserves unknown metadata JSON only after
native decoding succeeds.

A null row consumed by a Body-filter iterator is invalid; an empty object is
valid with missing attributes. Query-only ordinary iteration retains native
zero-model behavior for null rows. Cap/break can skip later selected null rows
or formatters after native whole-page decoding succeeds.

Body comparison precedes the common local status predicate. `WithStatus`
retains case-insensitive local matching; semantic Body `status` compares exact
JSON. Both ordinary and raw-filter adapters retain the existing removal of
raw `status` query values. Raw `WithQuery("status", ...)` alone is neither a
local predicate nor a server filter. Other raw query values, including
repetitions, reach the native builder unchanged. Native typed `API.List`
options keep their independent wire behavior.

`WithMaxItems` caps raw rows, `WithPaginated(false)` reads only the first page,
and consumer break stops iteration. These controls stop before reading a
continuation. Duplicate matching rows remain ordinary list rows; `All` returns
nil with terminal HTTP/decode/context errors rather than partial models.

The native pager accepts HTTP 200/204/300. A successfully parsed 204 page is
empty; an empty 204 advertised as `application/json` fails with native JSON EOF
before its emptiness check. Only top-level `next` supplies its continuation;
`links.next` is ignored. Foreign advertised URLs are followed, and failures
remain terminal. No same-origin/fixed-filter guard or synthetic marker/offset
fallback is added by this unit. The separate native Get/Create/Delete and
explicit-Ref policies remain available unchanged.

## OpenStackSDK comparison and evidence

Pinned source is OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`:
[`orders`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L233-L241),
[`Order` descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/order.py#L17-L58),
[`Resource.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2210-L2358),
and [`HREFToUUID`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_format.py#L18-L29).

Descriptor classification, raw attribute selection, and recursive matching are
implemented within the explicit Go policies above. Native typed decoding,
bool/number equality, selected-only formatting, and URL-parser boundaries
differ from Python's mutable Resource/property behavior. Python construction
and `to_dict` may format attributes that Go did not select. Generic inherited
cache, JMESPath, base-path/session/header controls, unknown-parameter controls,
and Python continuation/marker behavior are not claimed as complete parity.
Python may insert a wire limit from `max_items`; this native collection uses
only a local cap.

The seven
[`TestKeyManagerOrderListFilters` HTTP groups](../../../../api/keymanager_order_list_filters_test.go)
cover leaf/cached Connection paths, all fourteen properties, raw names and
metadata, ownership/collisions, native decoding, controls, and pagination.
The two separate
[`TestKeyManagerOrderIdentity` groups](../../../../api/keymanager_order_identity_test.go)
prove fixed explicit-ID routes and unsupported Name operations. Two
[Connection groups](../../../../connection_keymanager_order_filters_test.go)
exercise the shared service facade and raw-name namespace. These are local
HTTP/transport tests, not live-cloud validation or a claim that a prior wrong
HTTP route was reproduced.
