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

```go
hostScope, err := service.Users.InInstance(ctx, resource.ID("instance-id"),
    users.WithHost("192.0.2.10"))
if err != nil { return err }
id, err := hostScope.ResolveID(ctx, resource.Name("alice@office"))
if err != nil { return err }
user, err := hostScope.Get(ctx, id)
if err != nil { return err }
fmt.Println(user.Name, user.Host)
if err := hostScope.Delete(ctx, resource.ID(id)); err != nil { return err }
if err := hostScope.WaitDeleted(ctx, resource.ID(id)); err != nil { return err }
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/users"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListUsers(ctx context.Context, conn *sdk.Connection, instanceID, host string) error {
	service, err := conn.DatabaseV1(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Users.InInstance(ctx,
		resource.ID(instanceID), users.WithHost(host))
	if err != nil {
		return err
	}
	for user, err := range scope.List(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(user.Name, user.Host, user.Databases)
	}
	return nil
}
```

## References

- [Common](https://github.com/openstack/trove/blob/master/trove/extensions/common/common.py#L76-L85)
- [Wsgi](https://github.com/eventlet/eventlet/blob/master/eventlet/wsgi.py#L789-L791)
- [Common](https://github.com/openstack/python-troveclient/blob/master/troveclient/common.py#L32-L38)
- [..](../README.md)
- [Trove Users Contracts Test](../../../api/trove_users_contracts_test.go)
- [Listing](../../../docs/listing.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
