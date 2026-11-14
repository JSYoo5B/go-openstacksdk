# Acceleratorrequests

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
accel, err := conn.Accelerator(ctx)
if err != nil { return err }
created, err := accel.AcceleratorRequests.Create(ctx,
    acceleratorrequests.CreateOpts{DeviceProfileName: "fpga-profile"},
    request.Header("X-Custom", "example"))
if err != nil { return err }
for _, arq := range created.Requests {
    fmt.Println(arq.UUID, arq.State, arq.DeviceProfileGroupID)
}
```

## References

- [Cyborg API reference](https://docs.openstack.org/api-ref/accelerator/)
- [Arqs](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/arqs.py)
- [Accelerator Request](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/accelerator_request.py)
- [..](../README.md)
