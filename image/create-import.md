# Create Import

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_create_import1_test.go` in [example_create_import1_test.go](example_create_import1_test.go), compiled by `go test`.

```python
image = conn.image.create_image(
    "worker", data=data, disk_format="qcow2", container_format="bare",
    visibility="private", use_import=True, import_method="glance-direct",
    allow_duplicates=True, disable_vendor_agent=False, size=size)
```

See `example_create_import2_test.go` in [example_create_import2_test.go](example_create_import2_test.go), compiled by `go test`.

```python
image = conn.image.create_image(
    "worker-from-web", use_import=True, import_method="web-download", uri=uri,
    disk_format="qcow2", container_format="bare", visibility="private",
    stores=["fast", "reliable"], all_stores_must_succeed=False,
    allow_duplicates=True, disable_vendor_agent=False)
```

## References

- [Proxy.create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L178-L438)
- [_upload_image_put](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L727-L822)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html)
- [Stage](v2/imagedata/README.md)
- [Import](v2/imageimport/README.md)
- [Service.Upload](README.md)
- [Create Import Contracts Test](create_import_contracts_test.go)
- [Create Import Core Test](create_import_core_test.go)
- [Create Import Options Test](create_import_options_test.go)
- [Stage Prepare Test](v2/imagedata/stage_prepare_test.go)
- [Import Prepare Test](v2/imageimport/import_prepare_test.go)
