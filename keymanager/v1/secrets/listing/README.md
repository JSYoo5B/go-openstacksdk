# Secret list filters

`secrets.API.Resources.List` and `All` accept library-owned semantic filters.
Known query attributes become server parameters; known Body attributes match
the original JSON list row locally. Results remain native `secrets.Secret`
models. Raw filtering does not add missing fields or response metadata to that
native model, and it does not retrieve secret payloads.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/keymanager/v1/secrets"
    "gophercloudsdk/resource"
)

func listPlainSecrets(ctx context.Context, client *gophercloud.ServiceClient) ([]*secrets.Secret, error) {
    return secrets.New(client).Resources.All(ctx,
        resource.WithFilter("algorithm", "aes"),
        resource.WithFilter("content_types", map[string]any{"default": "text/plain"}),
        resource.WithMaxItems(20))
}
```

The algorithm filter sends `alg=aes`. The content-types filter performs a local
recursive object-subset comparison. `MaxItems` counts raw rows before local
filtering, so the result may contain fewer than 20 matching models. It does not
insert a server `limit` into this native pager.

## Query attributes

| Semantic attribute | Wire query |
| --- | --- |
| `acl_only` | `acl_only` |
| `algorithm` or `alg` | `alg` |
| `bits` | `bits` |
| `created` | `created` |
| `expiration` | `expiration` |
| `limit` | `limit` |
| `marker` | `marker` |
| `mode` | `mode` |
| `name` | `name` |
| `secret_type` | `secret_type` |
| `sort` | `sort` |
| `updated` | `updated` |

`WithFilters` applies Python's bulk canonical-name precedence: if both
`algorithm` and `alg` are present, `algorithm` wins even when its value is nil,
false, or empty. Individual `WithFilter` calls targeting the same wire key use
the last call. Strings, bools, and JSON numbers become scalar query values;
flat scalar arrays retain repeated values, omitting null elements. A nil value
or empty array retains key presence for collision checks but emits no URL
value. Known query values that are objects or nested arrays are invalid.

Unknown semantic keys are discarded, including values that cannot be encoded
as JSON. `status` is a Body attribute, while unknown `prefixlen`, `offset`, and
`list_base_path` are not added to the server query. Source control names
`max_items`, `paginated`, `base_path`, `allow_unknown_params`, `headers`,
`microversion`, `jmespath_filters`, `resource_type`, and `session` require their
dedicated supported APIs and are rejected as semantic filters.

A semantic `name` is a server filter; it does not enable the local name
predicate. Use `WithName` for that predicate. Semantic query values conflict
with raw `WithQuery`, `WithPageSize`, or the automatic name hint from `WithName`
when they target the same wire key, even when their values are equal.

## Local Body attributes

| Filter attribute | Original row value |
| --- | --- |
| `id` | Literal `id` when present; otherwise the full `secret_ref`. |
| `bit_length` | `bit_length` |
| `content_types` | `content_types` |
| `expires_at` | `expiration` |
| `created_at` | `created` |
| `updated_at` | `updated` |
| `secret_ref` | `secret_ref` |
| `secret_id` | Passive final component of `secret_ref`. |
| `status` | `status` |
| `payload` | `payload` |
| `payload_content_type` | `payload_content_type` |
| `payload_content_encoding` | `payload_content_encoding` |

These twelve attribute names are also the only supported explicit
`WithBodyFilter` keys. For example, semantic `created` is a query attribute,
while `created_at` is local; explicit `WithBodyFilter("created", ...)` is
invalid. A semantic Body filter and explicit Body filter for the same attribute
conflict. A Body filter and raw query with the same text remain independent.

Missing and JSON null values match a nil filter; an empty array is distinct.
Arrays compare exact length, order, and nested values. Objects use recursive
subset matching: an empty actual object fails an object filter, while an empty
filter matches a nonempty actual object. Decimal numbers compare exactly without
float64 conversion, including large integers and equivalent decimal/exponent
spellings. Strings are not coerced to numbers. Go deliberately distinguishes
bools from numbers, unlike Python's `True == 1`, and an object filter against a
nonobject is a nonmatch rather than a Python attribute-access error.

The three timestamp attributes compare original response strings. They do not
compare the parsed `time.Time` values in the native model. `id` also preserves
original JSON type and key presence: a literal null, empty, numeric, or object
ID prevents fallback to `secret_ref`. This passive matching never routes a
request using a response ID or reference.

`secret_id` is separate from `id`. It requires a reference with a scheme,
authority, and path, then returns the final original path component. It does
not validate a UUID, decode percent escapes, clean the path, or follow the URL.
Literal Unicode/space and existing `%xx` spellings are retained; query and
fragment are excluded, and a trailing slash produces an empty string.
Missing/null references bypass formatting. Only a selected `secret_id` invokes
the formatter, and an invalid selected reference returns an error before local
name/status predicates. Go's URL parser rejects some inputs accepted by
Python's `urlsplit`, such as a nonnumeric authority port; this is an explicit
parser boundary, not full URL-parser parity.

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/keymanager/v1/secrets"
    "gophercloudsdk/resource"
)

func listExactSecretRows(ctx context.Context, client *gophercloud.ServiceClient) ([]*secrets.Secret, error) {
    return secrets.New(client).Resources.All(ctx,
        resource.WithFilters(map[string]any{
            "mode": []any{"cbc", "gcm"},
            "bit_length": json.Number("256"),
            "created_at": "2026-10-02T03:04:05",
        }),
        resource.WithBodyFilter("status", "ACTIVE"),
        resource.WithPaginated(false))
}
```

Options snapshot JSON inputs at construction and application, including nested
maps/slices, and can be reused concurrently after configuring the client.
`WithFilters` replaces only the semantic namespace; nil/empty maps clear it
without clearing explicit Body filters or raw queries. Only final selected
semantic values are validated, so a later replacement or clear can remove an
earlier invalid semantic value. Existing explicit Body-option errors remain
sticky.

## Native decoding and pagination boundaries

The native decoder validates the whole current page before raw selectors run.
Its `bit_length` is an int, `content_types` is `map[string]string`, references
and declared strings are typed, and timestamps require the native format.
Malformed known fields anywhere in that page remain terminal, even beyond a
raw cap. The raw lane can preserve exact matching for fields the native model
does not expose, but cannot undo a native decoding failure. A consumed null
list row is rejected as invalid; an empty object remains a valid row with
missing attributes. Cap/break can avoid consuming a later null row or selected
formatter, after native whole-page decoding succeeds.

Body filters run before the common local name/status predicates. `WithStatus`
keeps its existing local case-insensitive behavior; semantic Body `status`
compares exact JSON. Ordinary Secrets `Resources` retains its existing removal
of raw `status` query values in both native and raw-filter lanes. Thus raw
`WithQuery("status", ...)` alone is not a local predicate or server filter.
Other raw query values, including repeats, reach the native builder unchanged.
The separate typed `API.List` options keep their existing wire behavior.

`WithMaxItems` caps raw rows; `WithPaginated(false)` consumes only the first
page; breaking `List` stops iteration. Cap, first-page mode, and consumer break
stop before reading a continuation. `All` returns nil with a terminal error
rather than returning partial rows. Context cancellation, native decode/HTTP
causes, and cycle errors remain observable.

The ordinary pager retains native HTTP 200/204/300 acceptance, with a successfully
parsed HTTP 204 page treated as empty. Native response parsing still happens
first: an empty 204 advertised as `application/json` returns a JSON EOF error.
It follows the native top-level `next` field, including a
foreign advertised URL; it does not add the same-origin/fixed-filter offset
guards used by the separate manual Finder. A foreign request failure is still
terminal. No UUID/offset/marker fallback is synthesized by this unit.
`Get`, [`Fetch`](../README.md), [`FindIdentity`](../finding/README.md), and
explicit-Ref lookup keep their own contracts.

## OpenStackSDK comparison

Source is pinned to OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`:
[`secrets`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L335-L343),
[`Secret` descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/secret.py#L22-L89),
[`Resource.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2210-L2358),
and [`HREFToUUID`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_format.py#L18-L29).

Query/Body classification, bulk alias precedence, raw attribute selection, and
recursive subset comparisons are implemented within the explicit Go policies
above. Native typed decoding, Go bool/number equality, selected-only formatting,
and parser boundaries differ from Python's mutable Resource/property handling.
Python construction and `to_dict` can format attributes that Go did not select.
Generic inherited cache, JMESPath, alternate base paths, session/headers,
unknown-parameter controls, and Python continuation/marker behavior are not
claimed as complete parity. Python may insert a limit from `max_items`; this
native Go collection intentionally uses only a local cap.

The eight
[`TestKeyManagerSecretListFilters` groups](../../../../api/keymanager_secret_list_filters_test.go)
use local HTTP servers and custom transports to verify query classification,
all twelve raw attributes, passive identities, exact JSON, ownership/collisions,
native decoding, controls, pagination, and unchanged Fetch/Finder behavior.
They are not live-cloud tests.
