# Create

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_create` in [example_create_test.go](example_create_test.go), compiled by `go test`.

```python
conn.object_store.create_object("backups", "empty.bin", data=b"")
conn.object_store.create_object("backups", "archive.tar", filename=local_filename)
conn.object_store.create_object(
    "backups", "dynamic.tar", filename=local_filename, use_slo=False
)
stale = conn.object_store.is_object_stale("backups", "archive.tar", local_filename)
```

## References

- [Info](../info.md)
- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L398-L529)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L691-L909)
- [FileSegment.seek/reset](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L448-L482)
- [Swift PUT API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L153-L299)
- [Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L173-L281)
- [SLO validation/finalization](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1341-L1612)
- [DLO validation](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/dlo.py#L422-L461)
- [Create core](create_core_test.go)
- [Create options](create_options_test.go)
- [Create Upload Test](create_upload_test.go)
- [Create Contracts Test](create_contracts_test.go)
- [stale core](stale_core_test.go)
- [stale options](stale_options_test.go)
- [Stale Contracts Test](stale_contracts_test.go)
- [Connection](../../../connectiontest/connection_objectstorage_create_object_test.go)
- [generator](../../../internal/cmd/sdkgen/swift_object_create_test.go)
