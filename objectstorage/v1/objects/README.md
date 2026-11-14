# Objects

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `ExampleAPI_InContainer` in [example_scope_test.go](example_scope_test.go), compiled by `go test`.

See `example_objects1_test.go` in [example_objects1_test.go](example_objects1_test.go), compiled by `go test`.

See `example_objects2_test.go` in [example_objects2_test.go](example_objects2_test.go), compiled by `go test`.

```python
obj = conn.object_store.get_object_metadata(
    "docs/readme ?.txt", container="assets"
)
same = conn.object_store.set_object_metadata(obj, owner="sdk", empty="")
assert same is obj
conn.object_store.delete_object_metadata(obj, keys=["owner"])
refreshed = conn.object_store.get_object_metadata(obj)
print(refreshed.metadata)
```

## Limitations

- Reads, uploads and waits report what one request observed. Swift replicas, eventual consistency, proxy caching and middleware decide when other clients see a change; a `HEAD` 404 or a matching metadata value is not a lock or a cluster-wide guarantee.
- Segmented uploads cannot be published atomically or rolled back. Unique segment prefixes and the returned cleanup evidence show what was written.
- `StreamObject` returns an `io.ReadCloser`: close it, and read it from one goroutine at a time.
- Text reads decode only UTF-8, a byte-order mark or a supported declared charset. Undeclared non-UTF-8 text returns `ErrUnsupported` with the original bytes; see [no charset guessing](../../../docs/sdk-support-ledger.md#deliberate-runtime-differences).

## References

- [`copy_object`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L531-L533)
- [Test Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/tests/unit/object_store/v1/test_proxy.py#L251-L252)
- [Listing](../../../docs/listing.md)
- [Containers](../containers/README.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
- [Obj](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L222)
- [Storage Object Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L631)
- [Server](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/obj/server.py#L665)
- [Metadata Contracts Test](metadata_contracts_test.go)
- [atomic header projection](metadata_contracts_test.go)
