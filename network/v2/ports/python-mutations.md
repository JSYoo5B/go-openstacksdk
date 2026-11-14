# Python Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
conn.network.create_port(network_id=network_id, is_admin_state_up=False, security_group_ids=[])
conn.network.update_port(port_id, name="", binding_host_id="", if_revision=0)
conn.network.update_port(resource_id)
conn.network.update_port(resource_id, if_revision=0)
```

See `Example_mutations` in [example_mutations_test.go](example_mutations_test.go), compiled by `go test`.
