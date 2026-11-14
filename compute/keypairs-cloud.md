# Keypairs Cloud

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
all_keys = conn.list_keypairs()
work_keys = conn.search_keypairs("work-*", filters={"type": "ssh"})
key = conn.get_keypair("workstation", user_id="owner-user-id")
created = conn.create_keypair("workstation", public_key="ssh-ed25519 ...")
deleted = conn.delete_keypair("workstation")
```

See `Example_keypairsCloud` in [example_keypairs_cloud_test.go](example_keypairs_cloud_test.go), compiled by `go test`.

## References

- [Keypairs List Find](keypairs-list-find.md)
- [Testing](../docs/testing.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
