# Rules

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
rule = conn.network.create_metering_label_rule(
    metering_label_id=label.id, direction="egress",
    destination_ip_prefix="0.0.0.0/0", is_excluded=False)
for item in conn.network.metering_label_rules(metering_label_id=label.id):
    print(item.direction, item.destination_ip_prefix)
conn.network.delete_metering_label_rule(rule)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3005-L3157)
- [`metering_label_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label_rule.py)
- [`labels`](../labels/README.md)
