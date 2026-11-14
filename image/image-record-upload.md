# Image Record Upload

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
with open("ubuntu.qcow2", "rb") as source:
    record = conn.image.upload_image(
        container_format="bare",
        disk_format="qcow2",
        data=source,
        name="ubuntu-example",
        visibility="private",
        filename="ubuntu.qcow2",
    )
print(record.id, record.status)
```

```sh
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example -service -size 0
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example -infer-size=false
```

See `Example_imageRecordUpload` in [example_image_record_upload_test.go](example_image_record_upload_test.go), compiled by `go test`.

## References

- [Image Record Waits](image-record-waits.md)
- [Install](../docs/install.md)
- [Upload Image](upload-image.md)
- [Create Import](create-import.md)
- [Implementation Plan](../docs/implementation-plan.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
