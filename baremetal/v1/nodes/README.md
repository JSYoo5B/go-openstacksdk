# Nodes

`Nodes.GetVirtualMedia` returns SDK-owned VirtualMedia with image, insertion state, and media types. Node record APIs have their own microversion discovery/selection controls; consult their operation guides instead of inferring behavior from native Get/List. The selected version and actual response remain distinct from advertised maximum versions.

Use the concrete API and operation-specific options in this package. Parent scopes fix the caller-selected resource; optional request values and missing-resource behavior are defined per operation.

Consult the [support ledger](../../../docs/sdk-support-ledger.md) and [operation reviews](../../../api/sdk_reviews.json) for supported behavior and unresolved boundaries.

## API reference

- [api_generated.go](api_generated.go)
- [python_list_records.go](python_list_records.go)
- [python_list_records_profile.go](python_list_records_profile.go)
- [python_list_records_proxy.go](python_list_records_proxy.go)
- [resources_generated.go](resources_generated.go)
- [virtual_media.go](virtual_media.go)
- [python_record_reads.go](python_record_reads.go)

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"),
    sdk.WithMicroversion("baremetal", "1.93"))
if err != nil { return err }
service, err := conn.BareMetal(ctx)
if err != nil { return err }
media, err := service.Nodes.GetVirtualMedia(ctx, nodeID)
if err != nil { return err }
fmt.Println(media.Image, media.Inserted, media.MediaTypes)
```

## References

- [Source-shaped Node reads](python-record-reads.md)
