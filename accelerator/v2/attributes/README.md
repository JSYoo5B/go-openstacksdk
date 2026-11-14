# Attributes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
attribute = conn.accelerator.create_attribute(
    deployable_id=17, key="trait:CUSTOM_ACCEL", value="required",
)
for item in conn.accelerator.attributes(deployable_id=17, key="trait:CUSTOM_ACCEL"):
    print(item.uuid, item.value)
```

```go
func createAndList(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := attributes.New(client)
    _, err := api.Create(ctx, attributes.CreateOpts{
        DeployableID: 17, Key: "trait:CUSTOM_ACCEL", Value: "required",
    })
    if err != nil { return err }
    for item, err := range api.List(ctx,
        resource.WithQuery("deployable_id", "17"),
        resource.WithQuery("key", "trait:CUSTOM_ACCEL")) {
        if err != nil { return err }
        fmt.Println(item.UUID, item.Value)
    }
    return nil
}
```

## References

- [pinned openstacksdk proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/_proxy.py)
- [pinned Attribute resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/attribute.py)
- [pinned Cyborg controller](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/attributes.py)
- [Accelerator](https://docs.openstack.org/api-ref/accelerator/)
