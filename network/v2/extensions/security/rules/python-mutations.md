# Python Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
conn.network.create_security_group_rule(
    security_group_id=group_id, direction="ingress", ether_type="IPv4",
    protocol=None, port_range_min=0, remote_address_group_id="",
)

conn.network.create_security_group_rules([
    dict(security_group_id=group_id, direction="ingress", ether_type="IPv4", remote_address_group_id=""),
    dict(security_group_id=group_id, direction="egress", ether_type="IPv6", remote_address_group_id=None),
])
```

See `Example_mutations` in [example_mutations_test.go](example_mutations_test.go), compiled by `go test`.
