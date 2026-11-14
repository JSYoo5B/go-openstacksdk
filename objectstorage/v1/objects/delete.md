# Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_delete` in [example_delete_test.go](example_delete_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect()
conn.object_store.delete_object("report.txt", container="backups")
conn.object_store.delete_object("strict-report.txt", container="backups", ignore_missing=False)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L535-L561)
- [Object._raw_delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L374-L407)
- [Swift object API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L449-L625)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L455-L480)
- [DELETE default codes](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L582-L598)
- [SLO controller](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1801-L1829)
- [bulk outcome](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/bulk.py#L331-L467)
- [Delete Core Test](delete_core_test.go)
- [Delete Options Test](delete_options_test.go)
- [public HTTP·missing·native hook](delete_contracts_test.go)
- [Connection Objectstorage Object Delete Test](../../../connectiontest/connection_objectstorage_object_delete_test.go)
- [Swift Object Delete Test](../../../internal/cmd/sdkgen/swift_object_delete_test.go)
