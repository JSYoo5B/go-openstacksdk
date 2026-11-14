# Temp Url Key

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
conn.object_store.set_account_temp_url_key(account_secret)
conn.object_store.set_container_temp_url_key(
    "books", container_secret, secondary=True
)
key = conn.object_store.get_temp_url_key("books")
```

See `Example_tempUrlKey` in [example_temp_url_key_test.go](example_temp_url_key_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L923)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L68)
- [Tempurl](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/tempurl.py)
- [account core](accounts/temp_url_key_core_test.go)
- [options](accounts/temp_url_key_options_test.go)
- [container core](containers/temp_url_key_core_test.go)
- [options](containers/temp_url_key_options_test.go)
- [getter core](temp_url_key_core_test.go)
- [options](temp_url_key_options_test.go)
- [Temp Url Key Contracts Test](temp_url_key_contracts_test.go)
- [Connection Objectstorage Temp Url Key Test](../../connectiontest/connection_objectstorage_temp_url_key_test.go)
- [Swift Temp Url Key Test](../../internal/cmd/sdkgen/swift_temp_url_key_test.go)
