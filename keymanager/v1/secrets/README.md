# Secret metadata and payload composition

`Fetch` adds the composition used by OpenStackSDK's `get_secret`: it always
requests the secret's JSON representation and conditionally requests its payload.
The existing native `Get`, `GetPayload`, user-metadata operations, and `Resources`
contracts remain available.
`GetPayload` returns buffered `[]byte` and closes the native response before
returning; [payload retrieval and migration](payload.md) explains its separate
request and the corrected return type.

`FindIdentity(ctx, identity, options...)` also provides library-owned ID/name
lookup with conditional payload on direct success and metadata-only list
fallback. [Identity lookup comparison](finding/README.md) describes shared
options, nullable/full-reference matching, duplicate detection, and pagination.

`CreateRecord(ctx, options...)` returns the input-seeded Resource view and the actual POST response separately, without a metadata or payload GET. [Creation comparison](../metadata-create.md) explains the same attribute options used for Containers, Orders, and Secrets.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func fetchSecret(ctx context.Context, client *gophercloud.ServiceClient) (*secrets.FetchedSecret, error) {
    return secrets.New(client).Fetch(ctx, resource.ID("secret-id"))
}
```

The first request is `GET /secrets/{id}` with `Accept: application/json`.
If the exact response key `content_types` is absent, no payload request follows.
An object with `default:null` also skips payload. Otherwise its exact `default`
key must be a string; missing keys, null/nonobject containers, and other default
types return an error with the metadata response body, headers, and status.
An empty string is a selected value and sends an empty `Accept` header.

`WithFetchContentType` represents a preexisting Python `payload_content_type`
attribute: it wins over the response's default, including malformed
`content_types`. A response's `payload_content_type` does not select the payload
type for that call. `WithFetchPayload(false)` is an additional Go option that
skips selection and the payload request after reading metadata.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func fetchBinarySecret(ctx context.Context, client *gophercloud.ServiceClient) (*secrets.FetchedSecret, error) {
    value, err := secrets.New(client).Fetch(ctx, resource.ID("secret-id"),
        secrets.WithFetchContentType("application/octet-stream"),
        secrets.WithFetchHeader("X-Trace", "caller-trace"))
    // value may contain the successful metadata response even when err
    // describes a later selection, payload HTTP/read, or UTF-8 failure.
    return value, err
}
```

The second request uses the original fixed `GET /secrets/{id}/payload` URI.
Neither a returned `id` or `secret_ref`, nor a changed `ResourceBase` after the
metadata request, changes that URI. References in the response are never
followed. `SecretID` records the request identity separately; the metadata
`Body` contains only actual response fields and has no synthesized ID.

`FetchedSecret.Body` retains original JSON field values, unknown fields, null versus
omission, and numbers beyond floating-point precision. Declared typed fields
use exact canonical wire keys; case variants are preserved as raw extensions.
`BitLength` and `ContentTypes` remain raw JSON. Declared string fields reject
incompatible JSON types and collapse null/omission to their Go zero value; the
raw `Body` preserves that distinction. Timestamp pointers retain the original
string without timestamp parsing.

`Payload` owns separate bytes, selected `Accept`, response headers, and status.
Only the exact choice `text/plain` decodes strict UTF-8 into `Payload.Text`.
Parameterized types such as `text/plain;charset=utf-8` return bytes, and the
payload response's `Content-Type` never changes this decision. Invalid UTF-8
returns an accepted-response error while retaining the payload bytes and
evidence. Metadata headers and payload headers remain separate.

| Option | Behavior |
| --- | --- |
| `WithFetchOptions(FetchOpts)` | Snapshots the optional bool/string pointers at construction and application. |
| `WithFetchContentType(string)` | Overrides automatic selection, including an explicit empty string. |
| `WithFetchPayload(bool)` | Defaults to automatic retrieval; false requests only metadata. |
| `WithFetchHeader(key, value)` | Adds an extra header to both requests, subject to service-header precedence. |

Both requests require a configured `key-manager` client and share its provider,
current authentication, HTTP transport, retry policy, and context. The SDK
revalidates source configuration after options/name resolution and between the
requests. Options and request headers are copied before name lookup; owned
options can be reused concurrently after configuring the client. Conflicting
case variants of a header are rejected, while equal values are accepted.
Configured service headers retain native precedence over per-call extras.
`Accept`, authentication, routing, content, and version headers are owned by the
SDK and cannot be set through Fetch extras or `ServiceClient.MoreHeaders`.
Use `WithFetchContentType` to select the payload representation. Body, query,
and argument extensions are unsupported.

An explicit `resource.ID` performs no resolver HTTP. An explicit
`resource.Name` resolves once through the existing native Secrets collection,
then fixes the selected safe single-segment ID. This name convenience is an
explicit Go extension; it does not add heuristic ID/name fallback to the pinned
`get_secret` declaration. The native collection's decoder and lookup policy
still govern that optional lookup.

## OpenStackSDK comparison

Source is pinned to OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`:
[`get_secret`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L322-L333)
calls [`Secret.fetch`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/secret.py#L91-L138).
The published [Barbican secret reference](https://docs.openstack.org/barbican/latest/api/reference/secrets.html)
documents root JSON and payload GET success as HTTP 200.

The composition closes the metadata-first selection, fixed URI, and exact
UTF-8/bytes behavior for a stateless explicit Go reference. Go deliberately
requires HTTP 200 for both phases and preserves native HTTP/transport/context
errors. Python's custom fetch omits generic Resource response translation;
HTTP failures follow the configured session/adapter policy. Go never retries a
successful response because selection or decoding failed.

Python adds payload to a response dictionary and only then updates and cleans
the existing mutable resource. Its seed/cache may retain prior values, and its
untyped property descriptors accept values outside Go's declared string model.
Go returns an owned response snapshot, keeps actual metadata JSON separate from
payload, retains raw unknown fields, and returns partial metadata with an error
after a later failure. It does not implement mutable Resource cache/dirty
state, generic `base_path` overrides, or payload `skip_cache` controls. Native
`Get` keeps its own accepted statuses/model behavior; this new helper does not
change it. These remaining differences are explicit rather than a claim of
complete inherited Resource parity.

The declared `get_secret(secret)` metadata-plus-conditional-payload getter is
reviewed as a `go_mapping` with these explicit Go differences. Its native
neighbors, separate find/list declarations and inherited Resource lifecycle
retain their own support reviews; this closes only the named getter contract.

HTTP evidence is in
[keymanager_secret_fetch_test.go](../../../api/keymanager_secret_fetch_test.go),
covering the eight `TestKeyManagerSecretFetch` contract groups with local HTTP
servers and custom transports. This is not a live-cloud test.

## Native Get, List, Create, Update and Delete

`API.Get`, `API.List`, `API.Create`, `API.Update` and `API.Delete` call the
pinned Gophercloud `v2.15.0` [secret requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/keymanager/v1/secrets/requests.go)
unchanged. IDs are joined into `secrets/{id}` without escaping, so a slash or
query delimiter changes the native path.

| Method | Request | Accepted statuses |
|---|---|---|
| `Get(ctx, id)` | `GET secrets/{id}` | 200 |
| `List(ctx, options...)` | `GET secrets` with `ListOpts` query | native pager |
| `Create(ctx, opts, options...)` | `POST secrets` with the `CreateOpts` JSON body | 201 |
| `Update(ctx, id, opts, options...)` | `PUT secrets/{id}` with the raw payload body | 204 |
| `Delete(ctx, id)` | `DELETE secrets/{id}` | 202, 204 |

`Secret` decodes `created`, `updated` and `expiration` with the native
RFC3339-without-zone parser; a null expiration becomes the zero time. `Create`
omits empty fields, formats `Expiration` without a zone and sends `{}` for zero
options. `WithCreateField` adds extension JSON but rejects keys that are core
`CreateOpts` fields. The create response usually holds only `secret_ref`.

`Update` sends `UpdateOpts.Payload` as the raw body and maps `ContentType` and
`ContentEncoding` to headers. `WithUpdateHeader` adds other headers and rejects
the two core header names. `List` serializes `ListOpts` (including
`created`/`updated`/`expiration` date filters) plus `WithListQuery` values,
follows the body `next` link exactly as returned (host and path included),
stops on an empty `secrets` page and sends no further page after the caller
stops. Other statuses keep the native `gophercloud.ErrUnexpectedResponseCode`;
the SDK adds only `resource.OperationError{Resource: "secrets"}` context.

