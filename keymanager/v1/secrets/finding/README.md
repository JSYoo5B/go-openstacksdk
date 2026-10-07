# Finding a secret by identity

`secrets.API.FindIdentity` resolves one exact string ID or name and returns an
owned `FetchedSecret`. Existing explicit `resource.ID`/`resource.Name` APIs keep
their native contracts.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secrets"
)

func findSecret(ctx context.Context, client *gophercloud.ServiceClient) (*secrets.FetchedSecret, error) {
    return secrets.New(client).FindIdentity(ctx, "secret-id-or-name")
}
```

A safe unescaped single-segment identity first uses
[`Fetch`](../README.md): metadata `GET /secrets/{identity}`, then a conditional
`GET /secrets/{identity}/payload`. Both phases require HTTP 200. Metadata's exact
`content_types.default` selects payload, and only exact `text/plain` decodes
strict UTF-8. A successful direct result keeps its fixed request `SecretID`
separate from its actual metadata body. It is returned even if the response's
ID or name differs from the request.

Under the default compatible fallback policy, an actual HTTP 400, 403, or 404
from either Fetch phase permits a metadata-only list search. Accepted-response
decode/selection/UTF-8 failures, transport/read errors, and canceled contexts
are terminal. A fallback result comes from the list response and never triggers
another secret GET or payload GET. Its `Payload` is nil and its `SecretID` is
empty; request seeds and response references are not synthesized into it.

Names that cannot safely use an ID route go directly to the list. This includes
strings containing whitespace, slashes, or literal percent escapes. The string
is neither trimmed nor URL-decoded: `name%2Fpart` searches for that exact name.
Blank, invalid UTF-8, and control-containing input is rejected before HTTP.
The SDK automatically supplies the exact string as the server's `name` query.
Local matching still compares the original string against each row's ID **or**
canonical name, without turning the name query into a local name-only filter.

Response identity comparison is passive:

- An exact literal `id` key has priority, including null, empty, or nonstring
  values. Null/nonstring values do not match a string input and do not fall back
  to a reference. A row can still match through its name.
- If literal `id` is absent from a direct response, the original `SecretID` seed
  is available out of band. It never changes the payload request URI.
- If literal `id` is absent from a list row, its **full original** `secret_ref`
  is the alternate identity. The SDK does not extract a UUID, decode escapes,
  validate that reference as a route, or follow it. Different full references
  with the same final component remain different IDs.

All rows on all accepted continuation pages are observed before returning a
unique list match. A second ID-or-name match is ambiguous, including repeated
rows with the same ID. A late HTTP/decode/context error prevents returning an
earlier match. Metadata raw fields, unknown exact numbers, original timestamp
strings, and page headers/status are retained using the Fetch model's canonical
decoder. A null or scalar list row and incompatible declared typed fields fail
with the actual accepted page body, headers, and status.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secrets"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func findSecretStrict(ctx context.Context, client *gophercloud.ServiceClient) (*secrets.FetchedSecret, error) {
    return secrets.New(client).FindIdentity(ctx, "secret-id-or-name",
        resource.WithIdentityFindIgnoreMissing(false),
        resource.WithIdentityFindFallback(resource.FindFallbackNotFoundOnly))
}
```

| Policy | Behavior |
| --- | --- |
| Default `IgnoreMissing` | True; successful complete list absence returns nil, nil. |
| `WithIdentityFindIgnoreMissing(false)` | Complete absence returns a typed not-found error. |
| `FindFallbackCompatible` | Permits list fallback after actual HTTP 400, 403, or 404. |
| `FindFallbackNotFoundOnly` | Permits fallback only after HTTP 404. |
| `FindFallbackNever` | Uses direct Fetch only; rejects list-only names before HTTP. An actual 404 follows `IgnoreMissing`. |

List HTTP failures are never suppressed as logical absence. Query,
`Details`, `AllProjects`, and `GetExtraSpecs` options are unsupported, including
explicit false flags and present-but-empty query keys, on both direct and
list-only input paths. Source controls such as raw `max_items` query keys require
their own supported APIs and fail validation. User option callbacks run exactly
once per call; final bool pointers and maps are snapshotted before HTTP. Owned
options can be reused concurrently with a configured client.

The client must be a configured `key-manager` service. Each call snapshots its
service prefix, headers, selected version, and original provider pointer. The
provider's authentication/token, HTTP transport, and retry behavior remain
shared. Original source configuration is validated after options, before direct
Fetch and each list page, and after a successful direct Fetch. Configure service
client fields before concurrent calls; this does not add live monitoring of
those fields inside an already-running snapshot Fetch. Response references
never select a different provider or request route.

## OpenStackSDK comparison and pagination

Pinned source is OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`:
[`find_secret`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L302-L320)
accepts a string and `ignore_missing`, without query kwargs. It calls
[`Resource.find`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2491-L2582),
whose direct fetch includes the conditional payload step. The exception catch
surrounds that entire Fetch; fallback listing constructs metadata-only resource
objects and compares exact IDs or names across the iterator.

Go explicitly requires HTTP 200 and offers the three fallback policies above.
The pinned Python Proxy defaults to `raise_exc=False`, and Secret's custom Fetch
omits generic response translation, so a 4xx does not necessarily raise the
exceptions that Resource.find catches. Strict Go status handling and compatible
fallback are deliberate safety policies, not a claim that the default Python
Proxy takes the same branch for every response. Python's mutable seed/cache and
untyped property behavior also differ from Go's owned canonical typed/raw model.
Unlike standalone Fetch's useful partial metadata result, Finder returns nil
with an error on a terminal failure; HTTP evidence remains in the error.

The fallback starts with only the automatic name query: there is no invented
SDK limit/offset default, max-items shortcut, or UUID marker. A nonempty page
without an advertised continuation ends the search. An empty page also ends it.
Advertised offsets must move forward on the same origin and exact collection
path; the first continuation may introduce a positive limit, which stays fixed.
Omitted filters are inherited, and changed filters, backward/repeated offsets,
foreign origins, malformed links, or conflicting continuations fail. The SDK
supports body next/links and RFC HTTP `Link` headers under these guards.

These guards are intentional Go policies. Python's generic list implementation
may update query fields from next links and, once a limit is known, use a
truthy-limit marker fallback when no link exists. The direct `find_secret` has
no limit argument. Its generic HTTP Link branch accesses a literal `uri` key;
Go's RFC parser is an explicit improvement, not an assertion that that Python
branch accepts every ordinary requests link dictionary. Generic inherited
list/cache/base-path controls are not added to this direct declaration.

The eight
[`TestKeyManagerSecretFind` HTTP groups](../../../../api/keymanager_secret_find_test.go)
prove direct composition, fallback boundaries, passive raw identities, all-page
observations, option ownership, terminal errors, fixed source snapshots, and
preflight/native Ref isolation. They use local servers and custom transports,
not a live cloud.
