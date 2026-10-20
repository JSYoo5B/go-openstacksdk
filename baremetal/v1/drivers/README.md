# Drivers

`Drivers.Resources` uses Driver.Name for ID and name. Reads, lists, and exact name finding are available; deletion and status waits return ErrUnsupported. Driver capabilities are response data rather than evidence that every deployment enables them.

Use the concrete API and operation-specific options in this package. Parent scopes fix the caller-selected resource; optional request values and missing-resource behavior are defined per operation.

Consult the [support ledger](../../../docs/sdk-support-ledger.md) and [operation reviews](../../../api/sdk_reviews.json) for supported behavior and unresolved boundaries.

## API reference

- [api_generated.go](api_generated.go)
- [resources_generated.go](resources_generated.go)

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
service, err := conn.BareMetalV1(ctx)
if err != nil { return err }
driver, err := service.Drivers.Find(ctx, resource.ID("redfish"))
if err != nil { return err }
fmt.Println(driver.Name, driver.Hosts)
```

## References

- [Resources Generated](resources_generated.go)
- [Baremetal Named Resources Test](../../../api/baremetal_named_resources_test.go)
- [Bare Metal v1](../README.md)
