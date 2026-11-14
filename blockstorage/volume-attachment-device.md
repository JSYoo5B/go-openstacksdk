# Volume Attachment Device

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_volumeAttachmentDevice` in [example_volume_attachment_device_test.go](example_volume_attachment_device_test.go), compiled by `go test`.

```python
volume = {"attachments": [
    {"server_id": "server", "device": ""},
    {"server_id": "server", "device": "/dev/vdb"},
]}
device = conn.get_volume_attach_device(volume, "server")  # "": first match
```

## References

- [Reference](README.md)
- [..](../README.md)
- [cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L381-L396)
- [v2 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/volume.py#L44)
- [v3 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54)
- [field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L111)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v2/volumes/results.go#L11-L31)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/results.go#L12-L33)
