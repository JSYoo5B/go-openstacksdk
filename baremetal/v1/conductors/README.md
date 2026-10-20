# Conductors

`Conductors.Resources` uses Conductor.Hostname as both ID and name. Explicit IDs fetch directly; names require exact matching across the complete inventory. Reads and lists require selected microversion 1.49 or later; configure WithMicroversion or WithMicroversionRange explicitly. Calling the operation never raises the version automatically. Delete and status waits return ErrUnsupported.

Use the concrete API and operation-specific options in this package. Parent scopes fix the caller-selected resource; optional request values and missing-resource behavior are defined per operation.

Consult the [support ledger](../../../docs/sdk-support-ledger.md) and [operation reviews](../../../api/sdk_reviews.json) for supported behavior and unresolved boundaries.

## API reference

- [api_generated.go](api_generated.go)
- [resources_generated.go](resources_generated.go)

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.BareMetal, "1.49"))
if err != nil { return err }
service, err := conn.BareMetalV1(ctx)
if err != nil { return err }
conductor, err := service.Conductors.Find(ctx, resource.ID("conductor-01"))
if err != nil { return err }
fmt.Println(conductor.Hostname, conductor.Alive, conductor.Drivers)
```

## References

- [Baremetal](https://docs.openstack.org/api-ref/baremetal/)
- [Webapi Version History](https://docs.openstack.org/ironic/latest/contributor/webapi-version-history.html)
- [Conductor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/baremetal/v1/conductor.py)
- [Microversions](../../../docs/microversions.md)
- [Resources Generated](resources_generated.go)
- [Baremetal Named Resources Test](../../../api/baremetal_named_resources_test.go)
- [Bare Metal v1](../README.md)
