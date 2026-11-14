# Image Record Download

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
response = conn.image.download_image(
    "literal-image-id",
    output="ubuntu.qcow2",
    store_preferences=["fast", "archive"],
)
print(response.status_code)
```

```python
with open("ubuntu.qcow2", "wb") as output:
    response = conn.image.download_image("literal-image-id", output=output)

response = conn.image.download_image("literal-image-id")
data = response.content

response = conn.image.download_image("literal-image-id", stream=True)
try:
    with open("ubuntu.qcow2", "wb") as output:
        for chunk in response.iter_content(chunk_size=1048576):
            output.write(chunk)
finally:
    response.close()
```

```sh
go run . -cloud dev -id literal-image-id -mode file -output ubuntu.qcow2
go run . -cloud dev -id literal-image-id -mode writer -output ubuntu.qcow2 -service
go run . -cloud dev -id literal-image-id -mode buffer -prefer fast -prefer archive
go run . -cloud dev -id literal-image-id -mode stream -output ubuntu.qcow2 -timeout 10m
```

See `Example_imageRecordDownload1` in [example_image_record_download1_test.go](example_image_record_download1_test.go), compiled by `go test`.

See `ExampleService_DownloadImageRecord` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [Image Records](image-records.md)
- [Install](../docs/install.md)
- [Licensing](../docs/licensing.md)
- [`DownloadTo`](download.md)
- [Implementation Plan](../docs/implementation-plan.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
