# Members

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:image/v2/members.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/members.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/members.API.RawClient`

- Create posts {"member": memberID} to native images/{imageID}/members with JSON headers, service MoreHeaders and live token, decodes the native Member and accepts only 200.
  Tests: `image/v2/members/contracts_test.go:TestNativeImageMembersRoutesBodiesAndDecode`, `image/v2/members/contracts_test.go:TestNativeImageMembersStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native member call only; it is not Python member Resource parity, which the owned member record APIs cover.
- The SDK adds only request.Wrap operation context; native raw-ID ServiceURL construction, fixed OkCodes, retries and decoding are unchanged.

### `gophercloud:image/v2/members.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/members.API.Get`

- Get sends one GET to images/{imageID}/members/{memberID} joined from raw IDs without escaping, decodes the native Member and accepts only 200; rejections keep ErrUnexpectedResponseCode under Get/members context.
  Tests: `image/v2/members/contracts_test.go:TestNativeImageMembersRoutesBodiesAndDecode`, `image/v2/members/contracts_test.go:TestNativeImageMembersStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native member call only; it is not Python member Resource parity, which the owned member record APIs cover.
- The SDK adds only request.Wrap operation context; native raw-ID ServiceURL construction, fixed OkCodes, retries and decoding are unchanged.

### `gophercloud:image/v2/members.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/members.API.List`

- List lazily yields the single native members page in order, yields nothing for an empty array and returns native HTTP rejections from the iterator without extra operation context.
  Tests: `image/v2/members/contracts_test.go:TestNativeImageMembersRoutesBodiesAndDecode`, `image/v2/members/contracts_test.go:TestNativeImageMembersStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native member call only; it is not Python member Resource parity, which the owned member record APIs cover.
- The SDK adds only request.Wrap operation context; native raw-ID ServiceURL construction, fixed OkCodes, retries and decoding are unchanged.

### `gophercloud:image/v2/members.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/members.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/members.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/members.WithUpdateField`

- Update PUTs {"status": opts.Status}, including an empty status, appends non-colliding WithUpdateField extensions, rejects status collisions and nil options before HTTP, returns the decoded server Member and accepts only 200.
  Tests: `image/v2/members/contracts_test.go:TestNativeImageMembersRoutesBodiesAndDecode`, `image/v2/members/contracts_test.go:TestNativeImageMembersStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native member call only; it is not Python member Resource parity, which the owned member record APIs cover.
- The SDK adds only request.Wrap operation context; native raw-ID ServiceURL construction, fixed OkCodes, retries and decoding are unchanged.

### `gophercloud:image/v2/members.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/members.API.Delete`

- Delete sends DELETE to the raw member path and accepts only 204; other statuses keep the native error under Delete/members context without an owned receipt.
  Tests: `image/v2/members/contracts_test.go:TestNativeImageMembersRoutesBodiesAndDecode`, `image/v2/members/contracts_test.go:TestNativeImageMembersStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native member call only; it is not Python member Resource parity, which the owned member record APIs cover.
- The SDK adds only request.Wrap operation context; native raw-ID ServiceURL construction, fixed OkCodes, retries and decoding are unchanged.

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
- [Glance Policy Priorities](../../../docs/glance-policy-priorities.md)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/members/requests.go#L28-L86)
- [Delete](../images/delete.md)
