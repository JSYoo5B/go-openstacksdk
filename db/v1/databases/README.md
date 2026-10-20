# Databases

`Databases.InInstance(ctx, instanceRef)` fixes the parent instance once. Database names are child identifiers. Get/Find searches the fixed instance's inventory because pinned native APIs expose no per-database fetch. Create and CreateBatch send arrays and return only errors; accepted responses do not contain a created database object. Delete uses the fixed parent and literal database name. No status waiter is inferred from asynchronous acceptance.

Use the concrete API and operation-specific options in this package. Parent scopes fix the caller-selected resource; optional request values and missing-resource behavior are defined per operation.

Consult the [support ledger](../../../docs/sdk-support-ledger.md) and [operation reviews](../../../api/sdk_reviews.json) for supported behavior and unresolved boundaries.

## API reference

- [api_generated.go](api_generated.go)
- [scope.go](scope.go)

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
scope, err := service.Databases.InInstance(ctx, resource.Name("db-server"))
if err != nil { return err }

database, err := scope.Find(ctx, resource.Name("app"))
if err != nil { return err }
fmt.Println(database.Name, database.CharSet, database.Collate)

if err := scope.Create(ctx, databases.CreateOpts{
    Name: "logs",
    CharSet: "utf8mb4",
    Collate: "utf8mb4_unicode_ci",
}); err != nil { return err }
```

```go
if err := scope.CreateBatch(ctx, databases.BatchCreateOpts{
    {Name: "app"},
    {Name: "logs", CharSet: "utf8mb4"},
}); err != nil { return err }
```

```go
if err := scope.Delete(ctx, resource.ID("app")); err != nil { return err }
if err := scope.WaitDeleted(ctx, resource.ID("app")); err != nil { return err }
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListDatabases(ctx context.Context, conn *sdk.Connection, instanceID string) error {
	service, err := conn.DatabaseV1(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Databases.InInstance(ctx, resource.ID(instanceID))
	if err != nil {
		return err
	}
	databases, err := scope.All(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false))
	if err != nil {
		return err
	}
	for _, database := range databases {
		fmt.Println(database.Name, database.CharSet, database.Collate)
	}
	return nil
}
```

## References

- [..](../README.md)
- [Trove Databases Contracts Test](../../../api/trove_databases_contracts_test.go)
- [Listing](../../../docs/listing.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
