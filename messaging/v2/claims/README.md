# Zaqar claims

`claims.API` wraps Gophercloud v2.15.0 `messaging/v2/claims` for queue-scoped `/v2/queues/{queue}/claims` routes. Python openstacksdk Resource and workflow equivalence is tracked separately. Every method takes the queue name explicitly; there is no common `resource.Collection`.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST .../claims?limit=N` | 201, 204 | Body `ttl`/`grace` are sent only when non-zero; `Limit` is a query parameter; 204 returns no messages |
| `Get` | `GET .../claims/{id}` | 200 | Returns `*Claim` with its messages |
| `Update` | `PATCH .../claims/{id}` | 204 only | Body `ttl`/`grace` only when non-zero |
| `Delete` | `DELETE .../claims/{id}` | 204 only | Native 404 stays an HTTP response error |

`Create` accepts `WithCreateField` and `WithCreateQuery`; `Update` accepts `WithUpdateField`. A field extension that names `ttl` or `grace` fails with `resource.ErrInvalidOption` before any request; nil options fail the same way.

The native wrappers do not add a `Client-ID` header. Deployments that require one must receive it from the caller's client configuration.

A claim is a lease on messages: a 201 only means the server granted it. Cancelling a context stops the local call and does not release or roll back a remote claim; mutations are never replayed automatically.
