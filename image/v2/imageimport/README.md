# Imageimport

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_imageimport1_test.go` in [example_imageimport1_test.go](example_imageimport1_test.go), compiled by `go test`.

```python
image = conn.image.get_image(image_id)
response = conn.image.import_image(image)  # method='glance-direct'
```

See `example_imageimport2_test.go` in [example_imageimport2_test.go](example_imageimport2_test.go), compiled by `go test`.

```python
response = conn.image.import_image(
    image, method="web-download", uri=uri,
    stores=["fast", "reliable"], all_stores_must_succeed=False)
```

## References

- [Proxy.import_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L440-L536)
- [Image.import_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L368-L427)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#import-an-image)
- [Glance Import Contracts Test](../../../api/glance_import_contracts_test.go)
- [Import Options Test](import_options_test.go)
- [Import Core Test](import_core_test.go)
- [Glance Import Test](../../../internal/cmd/sdkgen/glance_import_test.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/imageimport/requests.go)
