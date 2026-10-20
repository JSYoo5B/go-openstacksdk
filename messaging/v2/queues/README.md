# Zaqar queues

`queues.API` wraps Gophercloud v2.15.0 `messaging/v2/queues` for the `/v2/queues` routes. Python openstacksdk Resource and workflow equivalence is tracked separately. This package has no common `resource.Collection`; every method is a native call.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `PUT /v2/queues/{name}` | 201, 204 | `QueueName` is the URL segment and is removed from the body; `EnableEncryptMessages` is a pointer so explicit `false` is sent |
| `Get` | `GET /v2/queues/{name}` | 200 | Body is the queue metadata; unknown keys are kept in `QueueDetails.Extra` |
| `GetStats` | `GET /v2/queues/{name}/stats` | 200 | Decodes the `messages` object into `Stats` |
| `List` | `GET /v2/queues` | 200, 204 | Follows `links` next entries; 204 ends iteration; `ListOpts` booleans send `true` only |
| `Delete` | `DELETE /v2/queues/{name}` | 204 | Native 404 stays an HTTP response error |
| `Purge` | `POST /v2/queues/{name}/purge` | 204 | Body `resource_types` is required |
| `Share` | `POST /v2/queues/{name}/share` | 200 | Returns the signed `QueueShare` |
| `Update` | `PATCH /v2/queues/{name}` | 200, 201, 204 | Sends `Content-Type: application/openstack-messaging-v2.0-json-patch` |

Extension inputs use `WithCreateField`, `WithPurgeField`, `WithShareField` and `WithListQuery`. A field extension that names a core input fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. `Update` accepts only `WithUpdateOptions`.

`Update` sends the `BatchUpdateOpts` patch list as the JSON array body (for example `[{"op":"replace","path":"/metadata/x","value":0}]`), preserving zero and false values. The pinned native call reflects over its builder argument and would send `{}` for the SDK builder, so the SDK serializes the list through `ToQueueUpdateMap` and keeps the native URL, content type, accepted codes and result. The upstream builder requires `op`, `path` and `value`: a nil `Value` (including on a `remove` operation) or an empty field fails locally with `Missing input` before any request, so pass a non-nil placeholder value for `remove`. `WithUpdateOptions` replaces the whole batch; there are no field extensions for the patch list, and an empty batch sends `[]`.

The native wrappers do not add a `Client-ID` header. Zaqar deployments that require one must receive it from the caller's client configuration.

Create, Delete, Purge and Update acknowledgements do not prove that later reads observe the change. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
