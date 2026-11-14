# Volume Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="source")
volume = conn.block_storage.find_volume(
    "data", ignore_missing=False, details=False, all_projects=True
)
print(volume.to_dict())
```

See `Example_volumeFind` in [example_volume_find_test.go](example_volume_find_test.go), compiled by `go test`.

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Volume Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)
