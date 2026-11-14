# Type Encryption Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
volume_type = conn.block_storage.find_type("fast")
qos = conn.block_storage.find_qos_spec("gold", project_id=project_id)

created = conn.block_storage.create_type_encryption(
    volume_type, provider="luks", control_location="front-end")
updated = conn.block_storage.update_type_encryption(
    volume_type=volume_type, key_size=512, cipher=None)
updated = conn.block_storage.update_type_encryption(
    encryption=updated, key_size=0)
```

See `Example_typeEncryption` in [example_type_encryption_test.go](example_type_encryption_test.go), compiled by `go test`.

## References

- [Policy](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)
- [Install](../../docs/install.md)
- [Python Encryption Contracts Test](volumetypes/python_encryption_contracts_test.go)
- [Python Find Records Test](volumetypes/python_find_records_test.go)
- [Python Find Records Test](qos/python_find_records_test.go)
