# Nova Images Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
images = conn.compute.images(details=False, min_disk=4, max_items=100)
image = conn.compute.get_image(image_id)
found = conn.compute.find_image(image_id)
metadata = conn.compute.fetch_image_metadata(image_id)
conn.compute.set_image_metadata(image_id, owner="ci")
conn.compute.delete_image_metadata(image_id, keys=["owner"])
```

See `Example_novaImages` in [example_nova_images_test.go](example_nova_images_test.go), compiled by `go test`.

## References

- [Compute](https://docs.openstack.org/api-ref/compute/#images-deprecated)
- [Routes](https://github.com/openstack/nova/blob/master/nova/api/openstack/compute/routes.py)
- [API Version Request](https://github.com/openstack/nova/blob/master/nova/api/openstack/api_version_request.py)
- [Install](../../docs/install.md)
- [Image Records Test](images/image_records_test.go)
