# Users

`Users.InInstance(ctx, instanceRef)` fixes the parent instance. User names and optional host selectors are distinct; Get/Find searches only that instance's inventory because pinned native APIs have no per-user fetch. Create/CreateBatch submit array bodies and return errors rather than fabricated created users. Preserve explicit host selection and the operation's deletion semantics. No status waiter is synthesized.

Use the concrete API and operation-specific options in this package. Parent scopes fix the caller-selected resource; optional request values and missing-resource behavior are defined per operation.

Consult the [support ledger](../../../docs/sdk-support-ledger.md) and [operation reviews](../../../api/sdk_reviews.json) for supported behavior and unresolved boundaries.

## API reference

- [api_generated.go](api_generated.go)
- [scope.go](scope.go)

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
scope, err := service.Users.InInstance(ctx, resource.Name("db-server"))
if err != nil { return err }
user, err := scope.Find(ctx, resource.Name("reader"))
if err != nil { return err }
fmt.Println(user.Name, user.Host)

if err := scope.Create(ctx, users.CreateOpts{
    Name: "writer",
    Password: password,
    Databases: databases.BatchCreateOpts{{Name: "app"}},
}); err != nil { return err }
```

```go
if err := scope.CreateBatch(ctx, users.BatchCreateOpts{
    {Name: "reader", Password: password, Host: "%"},
    {Name: "reader", Password: password, Host: "192.0.2.10"},
}); err != nil { return err }
```

See `Example_usersFragment3` in [example_users_fragment3_test.go](example_users_fragment3_test.go), compiled by `go test`.

See `example_users1_test.go` in [example_users1_test.go](example_users1_test.go), compiled by `go test`.

## References

- [Common](https://github.com/openstack/trove/blob/master/trove/extensions/common/common.py#L76-L85)
- [Wsgi](https://github.com/eventlet/eventlet/blob/master/eventlet/wsgi.py#L789-L791)
- [Common](https://github.com/openstack/python-troveclient/blob/master/troveclient/common.py#L32-L38)
- [..](../README.md)
- [Trove Users Contracts Test](../../../api/trove_users_contracts_test.go)
- [Listing](../../../docs/listing.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
