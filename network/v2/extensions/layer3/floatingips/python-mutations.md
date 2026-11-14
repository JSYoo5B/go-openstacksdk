# Python Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
conn.network.create_ip(floating_network_id=external_network_id, description="")
conn.network.update_ip(floating_ip_id, port_id=None, fixed_ip_address="", if_revision=0)
conn.network.update_ip(resource_id)
conn.network.update_ip(resource_id, if_revision=0)
```

See `Example_mutations` in [example_mutations_test.go](example_mutations_test.go), compiled by `go test`.
