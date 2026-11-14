# Image Record Import

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
response = conn.image.import_image(
    record,
    method="web-download",
    uri="https://images.example.org/image.qcow2",
    stores=["fast", "archive"],
    all_stores_must_succeed=False,
)
print(response.status_code)
```

```sh
go run . -cloud dev -image-id ID
go run . -cloud dev -image-id ID -method web-download -uri https://images.example.org/image.qcow2 -stores fast,archive -all-stores-must-succeed=false
go run . -cloud dev -image-id ID -service -method glance-download -remote-region RegionTwo -remote-image-id REMOTE_ID -remote-interface public
go run . -cloud dev -image-id ID -method copy-image -store archive
```

See `Example_imageRecordImport` in [example_image_record_import_test.go](example_image_record_import_test.go), compiled by `go test`.

## References

- [Image Record Stage](image-record-stage.md)
- [Install](../docs/install.md)
- [`API.ImageImport.ImportImage/ImportKnownImage`](v2/imageimport/README.md)
- [CreateAndImport](create-import.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L440-L536)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L368-L436)
- [Image Records](image-records.md)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L317-L390)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L330-L343)
