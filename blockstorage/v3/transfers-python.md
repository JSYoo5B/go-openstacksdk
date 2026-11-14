# Transfers Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="source")
transfer = conn.block_storage.create_transfer(
    volume_id="v1", name="handoff", no_snapshots=True
)
for candidate in conn.block_storage.transfers(details=False):
    print(candidate.id, candidate.name)
found = conn.block_storage.find_transfer("handoff", ignore_missing=False)
print(found.id)
```

See `Example_transfers` in [example_transfers_test.go](example_transfers_test.go), compiled by `go test`.

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Transfer Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/transfer.py)
