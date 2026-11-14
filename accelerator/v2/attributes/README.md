# Attributes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
attribute = conn.accelerator.create_attribute(
    deployable_id=17, key="trait:CUSTOM_ACCEL", value="required",
)
for item in conn.accelerator.attributes(deployable_id=17, key="trait:CUSTOM_ACCEL"):
    print(item.uuid, item.value)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [pinned openstacksdk proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/_proxy.py)
- [pinned Attribute resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/attribute.py)
- [pinned Cyborg controller](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/attributes.py)
- [Accelerator](https://docs.openstack.org/api-ref/accelerator/)
