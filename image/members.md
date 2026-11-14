# Members

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func shareAndReadMembers(ctx context.Context, owner, member *image.Service, parent resource.Ref, memberID string) (*image.ImageMemberAcknowledgement, error) {
    common := []image.ImageMemberOption{
        image.WithImageMemberOpts(image.ImageMemberOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        request.Header("X-Member-Request", "single"),
        image.WithImageMemberHeaders(map[string]string{"X-Example": "members"}),
    }
    created, err := owner.AddImageMember(ctx, parent, memberID, common...)
    if err != nil {
        return nil, err
    }
    fmt.Println("created response", created.StatusCode)
    current, err := owner.GetImageMember(ctx, parent, memberID, common...)
    if err != nil {
        return nil, err
    }
    fmt.Println("current response", current.StatusCode)
    updated, err := member.UpdateImageMember(ctx, parent, memberID, "accepted", common...)
    if err != nil {
        return nil, err
    }
    fmt.Println("updated response", updated.StatusCode)

    found, err := owner.FindImageMember(ctx, parent, memberID,
        image.WithFindImageMemberOpts(image.FindImageMemberOpts{}),
        request.Header("X-Member-Request", "find"),
        image.WithFindImageMemberHeaders(map[string]string{"X-Example": "members"}),
        request.IgnoreMissing(false))
    if err != nil {
        return nil, err
    }
    fmt.Println("found response", found.StatusCode)

    listOptions := []image.ListImageMembersOption{
        image.WithListImageMembersOpts(image.ListImageMembersOpts{}),
        request.Header("X-Member-Request", "list"),
        image.WithListImageMembersHeaders(map[string]string{"X-Example": "members"}),
        request.MaxItems(20),
    }
    for row, err := range owner.ListImageMembers(ctx, parent, listOptions...) {
        if err != nil {
            return nil, err
        }
        fmt.Println("member row fields", len(row.Body))
    }
    all, err := owner.AllImageMembers(ctx, parent, listOptions...)
    if err != nil {
        return nil, err
    }
    fmt.Println("collected members", len(all))

    ack, err := owner.RemoveImageMember(ctx, parent, memberID,
        image.WithRemoveImageMemberOpts(image.RemoveImageMemberOpts{}),
        request.Header("X-Member-Request", "remove"),
        image.WithRemoveImageMemberHeaders(map[string]string{"X-Example": "members"}),
        request.IgnoreMissing(true))
    return ack, err
}
```

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
