# Credentials Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
created = conn.identity.create_credential(
    user_id=user_id, type="ec2", project_id=project_id,
    blob='{"access":"ACCESS","secret":"SECRET"}')
credentials = conn.identity.credentials(user_id=user_id, type="ec2")
found = conn.identity.find_credential(credential_id)
```

See `Example_credentials` in [example_credentials_test.go](example_credentials_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Python Records Test](credentials/python_records_test.go)
- [Native Contracts Test](credentials/native_contracts_test.go)
