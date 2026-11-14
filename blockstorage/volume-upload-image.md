# Volume Upload Image

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_volumeUploadImage` in [example_volume_upload_image_test.go](example_volume_upload_image_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="devstack")
upload = conn.block_storage.upload_volume_to_image(volume_id, "new-image")
upload = conn.block_storage.upload_volume_to_image(
    volume_id, "", force=False, disk_format="", container_format="")
upload = conn.block_storage.upload_volume_to_image(
    volume_id, "new-image", visibility="", protected=False)
# The selected value need not be a dict or include image_id.
```

## References

- [Proxy.upload_volume_to_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1387-L1420)
- [Volume.upload_to_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L375-L403)
- [Discover](https://github.com/openstack/keystoneauth/blob/e759cf88c072d5bc7e21fc3fa884708ace9630af/keystoneauth1/discover.py#L433-L458)
- [Upload Image](../image/upload-image.md)
- [Volume Actions](volume-actions.md)
- [Volume image metadata](volume-image-metadata.md)
- [Block Storage API](README.md)
