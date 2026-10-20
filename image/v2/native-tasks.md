# Native Tasks

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:image/v2/tasks.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithCreateField`

- Create POSTs an envelope-free {type, input} body (required type, nil input sent as null) plus top-level extension fields to tasks, rejects a missing type, type/input extensions and nil options before HTTP, decodes the envelope-free task with RFC 3339 timestamps and accepts only 201.
  Tests: `image/v2/tasks/native_contracts_test.go:TestNativeTaskRoutesBodiesPagingAndDecode`, `image/v2/tasks/native_contracts_test.go:TestNativeTaskStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native image/v2/tasks call only; Python task Resource parity and the SDK-owned task waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:image/v2/tasks.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.API.Get`

- Get GETs tasks/{taskID}, decodes the envelope-free task (zero value for {}, nil without an error for a JSON null body, wrapped decode error for zone-less timestamps) and accepts only 200.
  Tests: `image/v2/tasks/native_contracts_test.go:TestNativeTaskRoutesBodiesPagingAndDecode`, `image/v2/tasks/native_contracts_test.go:TestNativeTaskStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native image/v2/tasks call only; Python task Resource parity and the SDK-owned task waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:image/v2/tasks.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.API.List`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithListQuery`

- List GETs tasks with limit, marker, sort_key, sort_dir and status (ListOpts.ID and ListOpts.Type carry json tags only and are never sent) plus extension query values, follows the body next path and query re-rooted under the client base without its host, stops on an empty next, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `image/v2/tasks/native_contracts_test.go:TestNativeTaskRoutesBodiesPagingAndDecode`, `image/v2/tasks/native_contracts_test.go:TestNativeTaskStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native image/v2/tasks call only; Python task Resource parity and the SDK-owned task waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/tasks/requests.go)
- [Imageimport](imageimport/README.md)
