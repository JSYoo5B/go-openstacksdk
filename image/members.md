# Members

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_members_test.go` in [example_members_test.go](example_members_test.go), compiled by `go test`.

```python
def share_and_read_members(owner_conn, member_conn, image_id, member_id):
    created = owner_conn.image.add_member(image_id, member=member_id)
    current = owner_conn.image.get_member(member_id, image_id)
    updated = member_conn.image.update_member(
        member_id, image_id, status="accepted"
    )
    rows = list(owner_conn.image.members(image_id))
    found = owner_conn.image.find_member(
        member_id, image_id, ignore_missing=False
    )
    removed = owner_conn.image.remove_member(
        member_id, image=image_id, ignore_missing=True
    )
    return created, current, updated, rows, found, removed
```

## References

- [Member Record Mutations](member-record-mutations.md)
- [Member Records](member-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1222-L1386)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)
- [Image Members](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_members.py#L234-L280)
- [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_members.py#L396-L435)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#list-image-members)
- [show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_members.py#L282-L312)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#create-image-member)
- [Members Contracts Test](members_contracts_test.go)
- [core tests](members_core_test.go)
- [option tests](members_options_test.go)
- [Connection test](../connectiontest/connection_image_members_test.go)
- [generator test](../internal/cmd/sdkgen/glance_members_test.go)
