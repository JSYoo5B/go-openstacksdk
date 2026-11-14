# Python Resource Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
ag = conn.network.create_address_group(name="web", addresses=[])
scope = conn.network.create_address_scope(name="scope", ip_version=4, is_shared=False)
pool = conn.network.create_subnet_pool(
    name="pool", prefixes=["10.60.0.0/16"], default_prefix_length=24,
    minimum_prefix_length=16, maximum_prefix_length=28, is_shared=False,
)
trunk = conn.network.create_trunk(port_id=parent_port_id, is_admin_state_up=False, sub_ports=[])
policy = conn.network.create_qos_policy(name="gold", is_shared=False, is_default=False)

unchanged = conn.network.update_address_group(ag.id)
conn.network.update_address_group(ag.id, description="")
conn.network.update_address_scope(scope.id, is_shared=False)
conn.network.delete_address_group(ag.id)
conn.network.delete_address_scope(scope.id, ignore_missing=False)
```

```go
err := api.DeleteWithOptions(ctx, id)
err = api.DeleteWithOptions(ctx, id, request.IgnoreMissing(false))
strict := false
option := addressgroups.WithDeleteOptions(addressgroups.DeleteOpts{IgnoreMissing: &strict})
```

See `Example_resourceMutations1` in [example_resource_mutations1_test.go](example_resource_mutations1_test.go), compiled by `go test`.

## References

- [Python Read Delete](python-read-delete.md)
