# Python Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
conn.network.create_network(name="", is_admin_state_up=False, subnet_ids=[])
conn.network.update_network(network_id, name="", is_router_external=False, if_revision=0)
conn.network.update_network(resource_id)
conn.network.update_network(resource_id, if_revision=0)
```

See `Example_mutations` in [example_mutations_test.go](example_mutations_test.go), compiled by `go test`.
