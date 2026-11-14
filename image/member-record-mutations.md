# Member Record Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

owner = openstack.connect(cloud="owner")
recipient = openstack.connect(cloud="recipient")
image_id = "image-id"
member_id = "recipient-project-id"

created = owner.image.add_member(image_id, member_id=member_id)
fetched = owner.image.get_member(member_id, image_id)
updated = recipient.image.update_member(fetched, image_id, status="accepted")
owner.image.remove_member(member_id, image_id, ignore_missing=False)
```

See `Example_memberRecordMutations` in [example_member_record_mutations_test.go](example_member_record_mutations_test.go), compiled by `go test`.

## References

- [Member Records](member-records.md)
- [Members](members.md)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L223-L292)
- [Install](../docs/install.md)
- [add_member 1222–1241](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1222-L1241)
- [get_member 1316–1336](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1316-L1336)
- [update_member 1356–1386](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1356-L1386)
- [remove_member 1243–1271](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1243-L1271)
- [Member descriptor 16–43](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/member.py#L16-L43)
- [generic proxy create/update/delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L669-L799)
- [Resource response translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394)
- [microversion negotiation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418-L1439)
- [create](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1519-L1638)
- [commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L1937)
- [delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094-L2152)
