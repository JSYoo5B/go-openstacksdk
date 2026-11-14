# Grouptypes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
gt = conn.block_storage.create_group_type(name="gold", is_public=True)
conn.block_storage.create_group_type_group_specs(gt, {"consistent_group_snapshot_enabled": "<is> True"})
conn.block_storage.delete_group_type(gt)
```

See `ExampleAPI_CreateGroupSpecs` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2809-L3025)
- [`group_type.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_type.py)
