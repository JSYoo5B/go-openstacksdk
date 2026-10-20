# Barbican native containers

`containers.New(client)` exposes the pinned Gophercloud `v2.15.0`
[container requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/keymanager/v1/containers/requests.go)
unchanged. A container groups secret references; consumers record which
services use it. IDs are joined into the path without escaping, so a slash or
query delimiter changes the native path.

```go
api := containers.New(client)
created, err := api.Create(ctx, containers.CreateOpts{
    Type: containers.GenericContainer, Name: "web-tls",
    SecretRefs: []containers.SecretRef{{Name: "cert", SecretRef: certRef}},
})
if err != nil { return err }
for value, err := range api.List(ctx, containers.WithListOptions(containers.ListOpts{Name: "web-tls"})) {
    if err != nil { return err }
    fmt.Println(value.ContainerRef, value.Status)
}
fmt.Println(created.ContainerRef)
```

| Method | Request | Accepted statuses | Result |
|---|---|---|---|
| `Get(ctx, id)` | `GET containers/{id}` | 200 | `Container` |
| `List(ctx, options...)` | `GET containers` with `ListOpts` | native pager | lazy `Container` rows |
| `Create(ctx, opts, options...)` | `POST containers` | 201 | `Container`, usually only `container_ref` |
| `Delete(ctx, id)` | `DELETE containers/{id}` | 202, 204 | error only |
| `CreateConsumer(ctx, id, opts, options...)` | `POST containers/{id}/consumers` | 200 | the updated `Container` |
| `DeleteConsumer(ctx, id, options...)` | `DELETE containers/{id}/consumers` with a JSON body | 200 | the updated `Container` |
| `ListConsumers(ctx, id, options...)` | `GET containers/{id}/consumers` | native pager | lazy `Consumer` rows |
| `CreateSecretRef(ctx, id, ref, options...)` | `POST containers/{id}/secrets` | 201 | `Container` |
| `DeleteSecretRef(ctx, id, options...)` | `DELETE containers/{id}/secrets` with a JSON body | 204 | error only |

`CreateOpts.Type` is required before HTTP and `Name` has no `omitempty`, so an
empty name is sent as `""`. Consumer bodies use the field names `name` and
`URL`; the consumer and secret-ref deletes send their bodies with the DELETE
request through `WithDeleteConsumerOptions` and `WithDeleteSecretRefOptions`.
Times decode with the native RFC3339-without-zone parser. The `With...Field`
options add extension JSON and reject keys already in the body.

Both lists serialize the container `ListOpts` (`limit`, `name`, `offset`) plus
extension query values and follow the body `next` link exactly as returned.
`ListConsumers` reuses the container `ListOpts` type, so a `Name` is sent too.
A list stops on an empty page and sends no further page after the caller
stops. Other statuses keep the native `gophercloud.ErrUnexpectedResponseCode`;
the SDK adds only `resource.OperationError{Resource: "containers"}` context.

## Python find (`FindIdentity`)

`FindIdentity(ctx, identity, options...)` maps the pinned Python `find_container`
proxy, which calls `Resource.find`. A safe identity first gets one strict
metadata `Fetch`. After a clean 400, 403 or 404 the SDK lists every page
without a name query (the resource maps no name filter) and matches rows by
their literal `id`, which for a list row is the full `container_ref` reference, or by a
string `name`. A bare UUID therefore matches only through the direct GET. One
match is returned, two are `resource.ErrAmbiguous`, and none returns
`nil, nil` unless `resource.WithIdentityFindIgnoreMissing(false)` asks for
`resource.ErrNotFound`. Other statuses stop without the list. An identity that
is not a safe path segment, such as a full reference, skips the direct GET.
Query, details and project options are rejected because the Python call takes
only `ignore_missing`.

