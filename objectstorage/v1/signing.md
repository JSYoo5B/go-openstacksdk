# Signing

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
key = b"replace-with-installed-secret"
expires, signature = conn.object_store.generate_form_signature(
    "reports",
    "incoming/",
    "https://example.com/upload-finished",
    64 * 1024 * 1024,
    3,
    300,
    temp_url_key=key,
)
temp_path = conn.object_store.generate_temp_url(
    "/v1/AUTH_project/reports/incoming/",
    300,
    "GET",
    prefix=True,
    iso8601=True,
    ip_range="192.0.2.0/24",
    temp_url_key=key,
)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L998-L1210)
- [FormPost](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/formpost.py#L389-L424)
- [TempURL](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/tempurl.py#L759-L788)
- [Formpost](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/formpost.py#L296-L310)
- [Temp Url Key](temp_url_key.md)
- [Form core](form_signature_core_test.go)
- [Form Signature Options Test](form_signature_options_test.go)
- [TempURL core](temp_url_core_test.go)
- [Temp Url Options Test](temp_url_options_test.go)
- [Temp Url Signing Contracts Test](temp_url_signing_contracts_test.go)
- [Connection Objectstorage Signing Test](../../connectiontest/connection_objectstorage_signing_test.go)
- [Swift Signing Test](../../internal/cmd/sdkgen/swift_signing_test.go)
