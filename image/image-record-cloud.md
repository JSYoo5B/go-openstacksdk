# Image Record Cloud

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
visible = conn.list_images()
everything = conn.list_images(show_all=True)
ubuntu = conn.search_images("ubuntu-*", filters={"visibility": "public"})
image = conn.get_image("ubuntu-24.04")  # find_image, ignore_missing=True
by_id = conn.get_image_by_id("4c1e7d5a-0b6f-4a28-9a5e-7b0f3c1d2e9f")
print(len(visible), len(everything), [i.id for i in ubuntu])
if image is not None:
    print(image.to_dict(), by_id.status)
```

See `Example_imageRecordCloud` in [example_image_record_cloud_test.go](example_image_record_cloud_test.go), compiled by `go test`.

## References

- [Image Records](image-records.md)
- [Install](../docs/install.md)
- [Image Record Download](image-record-download.md)
- [Image Record Properties](image-record-properties.md)
- [Image Record Delete](image-record-delete.md)
- [Image Record Create](image-record-create.md)
- [Cinder Proxy create_image](../blockstorage/volume-upload-image.md)
- [Cloud list_images·search_images·get_image·get_image_by_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L84-L166)
- [get_image_exclude·get_image_name·get_image_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L221-L246)
- [wait_for_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L248-L263)
- [download_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L168-L219)
- [update_image_properties](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L432-L445)
- [delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L265-L313)
- [create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L315-L430)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)
- [_filter_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)
