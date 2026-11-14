# Labels

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
label = conn.network.create_metering_label(name="web", is_shared=False)
for item in conn.network.metering_labels(project_id=project_id):
    print(item.name, item.is_shared)
conn.network.delete_metering_label(label)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L2861-L3002)
- [`metering_label.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label.py)
- [`rules`](../rules/README.md)
