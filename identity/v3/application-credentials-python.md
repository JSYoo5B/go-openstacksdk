# Application Credentials Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
credentials = conn.identity.application_credentials(user_id, name="ci", max_items=100)
created = conn.identity.create_application_credential(
    user_id, "ci", description="", unrestricted=False, roles=[{"name": "member"}])
found = conn.identity.find_application_credential(user_id, "ci")
rule = conn.identity.get_access_rule(user_id, rule_id)
# conn.identity.delete_application_credential(user_id, created.id)
# conn.identity.delete_access_rule(user_id, rule.id, ignore_missing=False)
```

See `Example_applicationCredentials` in [example_application_credentials_test.go](example_application_credentials_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Python Records Test](applicationcredentials/python_records_test.go)
