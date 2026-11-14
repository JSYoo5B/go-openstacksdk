# Upload Image

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
with open("image.qcow2", "rb") as source:
    created = conn.image.upload_image(
        container_format="bare",
        disk_format="qcow2",
        data=source,
        name="upload-example",
        visibility="private",
        tags=["example"],
    )
print(created.id, created.status)
```

See `Example_uploadImage` in [example_upload_image_test.go](example_upload_image_test.go), compiled by `go test`.

## References

- [pinned Python `Proxy.upload_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L583)
- [Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462)
- [pinned API-ref](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-data.inc#L14)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#upload-binary-image-data)
- [wire/evidence](upload_image_contracts_test.go)
- [preflight/snapshot](upload_image_contracts_test.go)
- [Upload Image Contracts Test](upload_image_contracts_test.go)
- [core](upload_image_core_test.go)
- [options](upload_image_options_test.go)
- [Connection](../connectiontest/connection_image_upload_test.go)
- [generator](../internal/cmd/sdkgen/glance_image_upload_test.go)
