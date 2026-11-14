# Containers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_containers1_test.go` in [example_containers1_test.go](example_containers1_test.go), compiled by `go test`.

```python
container = conn.object_store.get_container_metadata("books")
print(container.metadata.get("book"))
refreshed = conn.object_store.set_container_metadata(
    container, book="Moby Dick", note="literal %2F 雪"
)
conn.object_store.delete_container_metadata(refreshed, ["note"])
```

See `Example_containers2` in [example_containers2_test.go](example_containers2_test.go), compiled by `go test`.

```python
created = conn.object_store.create_container(
    "books-lifecycle", metadata={"owner": "ops", "note": "literal %2F 雪"}
)
conn.object_store.delete_container(created, ignore_missing=False)
conn.object_store.delete_container("books-lifecycle")
```

See `Example_containers3` in [example_containers3_test.go](example_containers3_test.go), compiled by `go test`.

## References

- [Listing](../../../docs/listing.md)
- [Objects](../objects/README.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
- [Swift Listing Contracts Test](../../../api/swift_listing_contracts_test.go)
- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L189)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L68)
- [Storage Container Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-container-services.inc#L240)
- [Base](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/base.py#L1953)
- [Server](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/container/server.py#L59)
- [Metadata Contracts Test](metadata_contracts_test.go)
- [Storage Container Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-container-services.inc#L117)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L154)
- [Db](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/db.py#L1015)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1519)
- [Lifecycle Core Test](lifecycle_core_test.go)
- [Lifecycle Options Test](lifecycle_options_test.go)
- [Lifecycle Contracts Test](lifecycle_contracts_test.go)
- [Connection Objectstorage Container Lifecycle Test](../../../connectiontest/connection_objectstorage_container_lifecycle_test.go)
- [Swift Container Lifecycle Test](../../../internal/cmd/sdkgen/swift_container_lifecycle_test.go)
