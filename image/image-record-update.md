# Image Record Update

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")

updated = conn.image.update_image("image-id", name="release-image")

same = conn.image.update_image(updated, name=updated.name)
assert same is updated
```

See `Example_imageRecordUpdate` in [example_image_record_update_test.go](example_image_record_update_test.go), compiled by `go test`.

## References

- [Update](update.md)
- [native images.Update](v2/images/update.md)
- [Image proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)
- [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716-L748)
- [Image Records](image-records.md)
- [Install](../docs/install.md)
- [Image Records Model](image_records_model.go)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1251-L1273)
- [Pyproject](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/pyproject.toml#L20)
- [Source packing·flatten](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1207-L1229)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L439-L463)
- [Image Record Update Test](image_record_update_test.go)
- [Image Record Update Integration Test](../connectiontest/image_record_update_integration_test.go)
- [Diff Test](../internal/jsonpatch/diff_test.go)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L213-L223)
- [commit/no-op](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1875-L1937)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
