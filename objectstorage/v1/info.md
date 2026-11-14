# Info

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_info` in [example_info_test.go](example_info_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect()
caps = conn.object_store.get_info()
print("Swift capability:", caps.swift)
print("SLO capability:", caps.slo)
print("default selection:", conn.object_store.get_object_segment_size(None))
print("large selection:", conn.object_store.get_object_segment_size(8 * 1024**3))
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L860-L919)
- [Info Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/info.py#L25-L96)
- [Swift `/info` API](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage_info.inc#L1-L46)
- [Info](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/info.py#L55-L108)
- [effective constraints](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/server.py#L359-L378)
- [Constraints](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/constraints.py#L28-L66)
- [Slo](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/slo.py#L1893-L1922)
- [Info Core Test](info_core_test.go)
- [Info Options Test](info_options_test.go)
- [public HTTP·native retry·fallback](info_contracts_test.go)
- [Connection Objectstorage Info Test](../../connectiontest/connection_objectstorage_info_test.go)
- [Swift Info Test](../../internal/cmd/sdkgen/swift_info_test.go)
