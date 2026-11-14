# Volume Image Metadata

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_volumeImageMetadata` in [example_volume_image_metadata_test.go](example_volume_image_metadata_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.set_volume_image_metadata(volume_id, hw_disk_bus="scsi")
conn.block_storage.set_volume_image_metadata(volume_id, **{})
conn.block_storage.delete_volume_image_metadata(volume_id, keys=["z", "a", "z"])
conn.block_storage.delete_volume_image_metadata(volume_id, keys=[])
# Pinned all-path calls Volume.delete_image_metadata, which loops ordinary metadata.
conn.block_storage.delete_volume_image_metadata(volume)
```

## References

- [Proxy set/delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1164-L1197)
- [Volume image metadata actions](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L189-L207)
- [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L196-L200)
- [Volumes](v3/volumes/README.md)
- [Volume Actions](volume-actions.md)
- [Reference](README.md)
