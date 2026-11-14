# Credential Update Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
updated = conn.identity.update_credential(
    credential_id,
    blob='{"access":"example","secret":"replace-me"}',
)
print(updated.id, updated.blob)
```

See `Example_credentialUpdate` in [example_credential_update_test.go](example_credential_update_test.go), compiled by `go test`.

## References

- [Credentials Python](credentials-python.md)
- [Install](../../docs/install.md)
- [Python Update Test](credentials/python_update_test.go)
