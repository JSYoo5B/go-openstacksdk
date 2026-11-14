# Update

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_update` in [example_update_test.go](example_update_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect(cloud="dev")
updated = conn.image.update_image("image-id", name="release-image", is_protected=False)
same = conn.image.update_image(updated, name=updated.name)
assert same is updated
```

## References

- [Update](../../update.md)
- [Image Records](../../image-records.md)
- [Install](../../../docs/install.md)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L260-L418)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L236-L248)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/results.go#L161-L183)
- [Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1875-L1937)
- [owned UpdateImageRecord](../../image-record-update.md)
- [Source pyproject](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/pyproject.toml#L15-L20)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L81-L116)
- [API policy](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L203-L229)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L657-L718)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L160-L171)
- [Contracts Test](../../../api/contracts_test.go)
- [Update Contracts Test](../../update_contracts_test.go)
- [Mutation Contracts Test](../../mutation_contracts_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
