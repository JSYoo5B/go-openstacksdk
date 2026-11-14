# Read

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_read` in [example_read_test.go](example_read_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect()

remembered = conn.object_store.get_object(
    "report.txt", container="backups", remember_content=True
)
print("remembered text characters:", len(remembered.data))

data = conn.object_store.download_object("report.txt", container="backups")
print("downloaded bytes:", len(data))

chunks = conn.object_store.stream_object(
    "report.txt", container="backups", chunk_size=32 * 1024
)
for chunk in chunks:
    print("chunk bytes:", len(chunk))
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L288-L396)
- [Swift GET API reference](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L16-L151)
- [Swob](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/swob.py#L1350-L1478)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/objects/requests.go#L103-L171)
- [Read Core Test](read_core_test.go)
- [Read Stream Test](read_stream_test.go)
- [concrete options](read_options_test.go)
- [Read Contracts Test](read_contracts_test.go)
- [Connection Objectstorage Object Read Test](../../../connectiontest/connection_objectstorage_object_read_test.go)
- [Swift Object Read Test](../../../internal/cmd/sdkgen/swift_object_read_test.go)
