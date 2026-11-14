# Python Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
conn.network.create_security_group(name="", stateful=False, is_shared=False)
conn.network.update_security_group(group_id, name="", stateful=False, if_revision=0)
conn.network.update_security_group(resource_id)
conn.network.update_security_group(resource_id, if_revision=0)
```

See `Example_mutations` in [example_mutations_test.go](example_mutations_test.go), compiled by `go test`.
