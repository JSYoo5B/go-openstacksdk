# Python Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
conn.network.create_subnet(network_id=network_id, ip_version=4, cidr="10.61.0.0/24", is_dhcp_enabled=False)
conn.network.update_subnet(subnet_id, gateway_ip=None, is_dhcp_enabled=False, if_revision=0)
conn.network.update_subnet(resource_id)
conn.network.update_subnet(resource_id, if_revision=0)
```

See `Example_mutations` in [example_mutations_test.go](example_mutations_test.go), compiled by `go test`.
