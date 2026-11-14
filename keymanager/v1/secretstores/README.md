# Secretstores

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_secretstores1_test.go` in [example_secretstores1_test.go](example_secretstores1_test.go), compiled by `go test`.

See `example_secretstores2_test.go` in [example_secretstores2_test.go](example_secretstores2_test.go), compiled by `go test`.

```python
stores = conn.key_manager.secret_stores(
    name="backend-pattern", secret_store_id="store-id", limit=10,
)
```

See `ExampleWithListFilter` in [example_test.go](example_test.go), compiled by `go test`.

See `Example_string` in [example_secretstores3_test.go](example_secretstores3_test.go), compiled by `go test`.

## References

- [Keymanager Secretstores Test](../../../api/keymanager_secretstores_test.go)
- [Store Backends](https://docs.openstack.org/barbican/wallaby/api/reference/store_backends.html)
- [Policy](https://docs.openstack.org/barbican/2024.1/configuration/policy.html)
- [Policy](https://docs.openstack.org/barbican/train/configuration/policy.html)
