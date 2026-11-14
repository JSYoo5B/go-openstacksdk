# Accounts

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
account = conn.object_store.get_account_metadata()
print(account.metadata)
conn.object_store.set_account_metadata(book="Moby Dick", note="literal %2F 雪")
conn.object_store.delete_account_metadata(["note"])
```

See `Example_accounts` in [example_accounts_test.go](example_accounts_test.go), compiled by `go test`.

## References

- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L119)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L71)
- [Storage Account Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-account-services.inc#L120)
- [Base](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/base.py#L1953)
- [Metadata Contracts Test](metadata_contracts_test.go)
