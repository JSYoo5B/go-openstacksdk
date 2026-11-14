# Defaultrules

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
rule = conn.network.create_default_security_group_rule(
    direction="ingress", ether_type="IPv4", protocol="tcp",
    port_range_min=22, port_range_max=22, used_in_default_sg=True)
for item in conn.network.default_security_group_rules(ether_type="IPv4"):
    print(item.id, item.port_range_min)
conn.network.delete_default_security_group_rule(rule)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.
