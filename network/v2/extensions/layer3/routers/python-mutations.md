# Python Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
conn.network.create_router(name="edge", is_admin_state_up=False, is_distributed=False)
conn.network.update_router(router_id, name="", external_gateway_info=None, if_revision=0)
conn.network.update_router(resource_id)
conn.network.update_router(resource_id, if_revision=0)
```

See `Example_mutations` in [example_mutations_test.go](example_mutations_test.go), compiled by `go test`.
