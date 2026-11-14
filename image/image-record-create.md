# Image Record Create

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.create_image(
    name="ubuntu-example",
    filename="ubuntu.qcow2",
    visibility="private",
    allow_duplicates=True,
)

record = conn.image.create_image(
    name="binary-example",
    data=b"image bytes",
    validate_checksum=True,
)

record = conn.image.create_image(
    name="import-example",
    filename="ubuntu.qcow2",
    use_import=True,
    import_method="glance-direct",
)
```

```sh
go run . -cloud dev -scenario direct -file ubuntu.qcow2 -name ubuntu-example
go run . -cloud dev -scenario bytes -file ubuntu.qcow2 -name binary-example -service
go run . -cloud dev -scenario import -file ubuntu.qcow2 -name import-example
go run . -cloud task-cloud -scenario task -file ubuntu.qcow2 -name task-example -wait=true
go run . -cloud dev -scenario metadata -name metadata-example
```

See `Example_imageRecordCreate` in [example_image_record_create_test.go](example_image_record_create_test.go), compiled by `go test`.

## References

- [create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L178-L436)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Implementation Plan](../docs/implementation-plan.md)
- [Upload](upload-image.md)
- [CreateAndImport](create-import.md)
- [UploadImageRecord](image-record-upload.md)
- [StageImageRecord](image-record-stage.md)
- [ImportImageRecord](image-record-import.md)
- [Install](../docs/install.md)
- [Source filename helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L50-L63)
- [Create](../objectstorage/v1/objects/create.md)
