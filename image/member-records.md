# Member Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
image_id = "image-id"

for member in conn.image.members(image_id):
    print(member.to_dict())

member = conn.image.find_member("project-id", image_id, ignore_missing=False)
print(member.to_dict())
```

See `Example_memberRecords` in [example_member_records_test.go](example_member_records_test.go), compiled by `go test`.

## References

- [Members](members.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1289-L1354)
- [Install](../docs/install.md)
- [Member](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/member.py#L16-L43)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2524-L2582)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L251-L278)
- [Member Records Test](member_records_test.go)
- [Member Record Find Test](member_record_find_test.go)
- [Falsey Next Test](../internal/rest/falsey_next_test.go)
- [Members Contracts Test](members_contracts_test.go)
