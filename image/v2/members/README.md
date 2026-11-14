# Members

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := service.API.Members
member, err := api.Create(ctx, imageID, projectID)
if err != nil { return err }
member, err = api.Update(ctx, imageID, projectID, members.UpdateOpts{Status: "accepted"})
if err != nil { return err }
for value, err := range api.List(ctx, imageID) {
    if err != nil { return err }
    fmt.Println(value.MemberID, value.Status)
}
if err := api.Delete(ctx, imageID, projectID); err != nil { return err }
```

## References

- [Members](../../members.md)
- [Member Records](../../member-records.md)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/members/requests.go#L28-L86)
- [Delete](../images/delete.md)
