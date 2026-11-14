# Download

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_download1_test.go` in [example_download1_test.go](example_download1_test.go), compiled by `go test`.

```python
response = conn.image.download_image(image_id, output=output, stream=True)
```

See `example_download2_test.go` in [example_download2_test.go](example_download2_test.go), compiled by `go test`.

```python
image = conn.image.find_image(name, ignore_missing=False)
response = conn.image.download_image(
    image, output=output, stream=True, chunk_size=64 * 1024,
    store_preferences=["fast", "archive", "fast"])
```

## References

- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#download-binary-image-data)
- [Proxy.download_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L919-L968)
- [DownloadMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/_download.py#L26-L183)
- [Proxy.request](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L209-L285)
- [ImageData.Download](v2/imagedata/README.md)
- [CreateAndImport](create-import.md)
- [Service.Upload](README.md)
- [Download Contracts Test](download_contracts_test.go)
- [Download Core Test](download_core_test.go)
- [Download Options Test](download_options_test.go)
- [Glance Download Test](../internal/cmd/sdkgen/glance_download_test.go)
