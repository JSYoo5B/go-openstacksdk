# Imagedata

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_imagedata1_test.go` in [example_imagedata1_test.go](example_imagedata1_test.go), compiled by `go test`.

```python
image = conn.image.get_image(image_id)
image = conn.image.stage_image(image, data=data)
```

See `example_imagedata2_test.go` in [example_imagedata2_test.go](example_imagedata2_test.go), compiled by `go test`.

```python
image = conn.image.stage_image(image, data=data, size=remaining_bytes)
```

## References

- [Proxy.stage_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L538-L581)
- [Image.stage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L325-L366)
- [get_file_size](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462-L491)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#stage-binary-image-data)
- [Imageimport](../imageimport/README.md)
- [Glance Stage Contracts Test](../../../api/glance_stage_contracts_test.go)
- [Stage Core Test](stage_core_test.go)
- [Stage Options Test](stage_options_test.go)
- [Glance Stage Test](../../../internal/cmd/sdkgen/glance_stage_test.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/imagedata/requests.go)
- [Image Record Download](../../image-record-download.md)
