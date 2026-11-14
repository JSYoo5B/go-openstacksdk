# Trusts Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
created = conn.identity.create_trust(
    trustor_user_id=trustor_id,
    trustee_user_id=trustee_id,
    project_id=project_id,
    is_impersonation=False,
    roles=[{"name": "member"}],
)
for trust in conn.identity.trusts(trustor_user_id=trustor_id):
    print(trust.id, trust.is_impersonation)
found = conn.identity.find_trust(created.id, ignore_missing=False)
conn.identity.delete_trust(created.id, ignore_missing=False)
```

See `Example_trusts` in [example_trusts_test.go](example_trusts_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Python Records Test](trusts/python_records_test.go)
