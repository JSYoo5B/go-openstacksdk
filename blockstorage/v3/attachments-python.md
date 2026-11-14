# Attachments Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
reserved = conn.block_storage.create_attachment(volume_id)
attached = conn.block_storage.create_attachment(
    volume_id, instance=server_id, connector={"host": "compute-1"}, mode="rw")
for attachment in conn.block_storage.attachments(volume_id=volume_id):
    print(attachment.id, attachment.attach_mode)
```

See `Example_attachments` in [example_attachments_test.go](example_attachments_test.go), compiled by `go test`.

## References

- [Attachment proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1495-L1551)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/attachment.py)
- [Install](../../docs/install.md)
- [Python Records Contracts Test](attachments/python_records_contracts_test.go)
