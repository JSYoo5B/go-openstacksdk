# Cinder Volume Attachment

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_cinderVolumeAttachment` in [example_cinder_volume_attachment_test.go](example_cinder_volume_attachment_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.attach_volume(volume_id, "/dev/vdb", instance=server_id)
conn.block_storage.attach_volume(volume_id, "", instance="", host_name="ignored")
conn.block_storage.detach_volume(volume_id, attachment_id)
conn.block_storage.detach_volume(volume_id, attachment_id, force=True,
                                 connector={"host": "", "multipath": False, "extra": None})
```

## References

- [Proxy attach/detach](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1241-L1280)
- [Volume.attach/detach](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L234-L271)
- [AttachVolume](attach-volume.md)
- [DetachVolume](detach-volume.md)
- [Volume Actions](volume-actions.md)
- [service API](v3/volumes/README.md)
