# Tags

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/tags.Add`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.API.Add`

- Add PUTs servers/{id}/tags/{raw tag} without a body and accepts only 201/204.
  Tests: `compute/v2/tags/native_contracts_test.go:TestNativeTagsRoutesBodiesAndDecode`, `compute/v2/tags/native_contracts_test.go:TestNativeTagsStrictStatusesCheckAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server-ID tag call only; the InServer scope adds microversion, tag and missing-policy validation documented separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/tags.Check`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.API.Check`

- Check GETs servers/{id}/tags/{raw tag}, returns true for 204, false with nil error for 404, false with the error for other statuses, and expects only 204.
  Tests: `compute/v2/tags/native_contracts_test.go:TestNativeTagsRoutesBodiesAndDecode`, `compute/v2/tags/native_contracts_test.go:TestNativeTagsStrictStatusesCheckAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server-ID tag call only; the InServer scope adds microversion, tag and missing-policy validation documented separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/tags.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.API.List`

- List GETs servers/{id}/tags, returns the tags array in server order and accepts only 200.
  Tests: `compute/v2/tags/native_contracts_test.go:TestNativeTagsRoutesBodiesAndDecode`, `compute/v2/tags/native_contracts_test.go:TestNativeTagsStrictStatusesCheckAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server-ID tag call only; the InServer scope adds microversion, tag and missing-policy validation documented separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/tags.ReplaceAll`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.API.ReplaceAll`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.WithReplaceAllOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.WithReplaceAllField`

- ReplaceAll PUTs {"tags":[...]} with extensions beside the array, rejects nil tags, a tags override or nil option before HTTP, returns the server-ordered tags and accepts only 200.
  Tests: `compute/v2/tags/native_contracts_test.go:TestNativeTagsRoutesBodiesAndDecode`, `compute/v2/tags/native_contracts_test.go:TestNativeTagsStrictStatusesCheckAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server-ID tag call only; the InServer scope adds microversion, tag and missing-policy validation documented separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/tags.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.API.Delete`

- Delete DELETEs servers/{id}/tags/{raw tag} and accepts only 204.
  Tests: `compute/v2/tags/native_contracts_test.go:TestNativeTagsRoutesBodiesAndDecode`, `compute/v2/tags/native_contracts_test.go:TestNativeTagsStrictStatusesCheckAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server-ID tag call only; the InServer scope adds microversion, tag and missing-policy validation documented separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/tags.DeleteAll`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/tags.API.DeleteAll`

- DeleteAll DELETEs servers/{id}/tags and accepts only 204.
  Tests: `compute/v2/tags/native_contracts_test.go:TestNativeTagsRoutesBodiesAndDecode`, `compute/v2/tags/native_contracts_test.go:TestNativeTagsStrictStatusesCheckAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server-ID tag call only; the InServer scope adds microversion, tag and missing-policy validation documented separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/tags"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx,
        sdk.WithMicroversionRange(sdk.Compute, "2.26", "2.100"),
    )
    if err != nil { panic(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { panic(err) }
    serverTags, err := service.Tags.InServer(ctx, resource.Name("web.prod"))
    if err != nil { panic(err) }

    if err := serverTags.Add(ctx, "managed"); err != nil { panic(err) }
    present, err := serverTags.Check(ctx, "managed")
    if err != nil { panic(err) }
    fmt.Println(serverTags.ServerID(), present)
    values, err := serverTags.List(ctx)
    if err != nil { panic(err) }
    fmt.Println(values)
    values, err = serverTags.Replace(ctx, []string{"managed", "role=db"})
    if err != nil { panic(err) }
    fmt.Println(values)
    if err := serverTags.Remove(ctx, "managed", tags.WithMissingError()); err != nil {
        panic(err)
    }
    if err := serverTags.RemoveAll(ctx); err != nil { panic(err) }
}
```

## References

- [Compute](https://docs.openstack.org/api-ref/compute/#server-tags-servers-tags)
- [Microversions](../../../docs/microversions.md)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/tags/requests.go)
