# Servergroups

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/servergroups.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithCreateField`

- Create POSTs {"server_group":{...}} to os-server-groups omitting empty fields, places extensions inside the envelope, rejects a missing name, a core-key extension collision and a nil option before HTTP, decodes the response and accepts only 200.
  Tests: `compute/v2/servergroups/native_contracts_test.go:TestNativeServerGroupsRoutesBodiesAndDecode`, `compute/v2/servergroups/native_contracts_test.go:TestNativeServerGroupsStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native servergroups call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/servergroups.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.Get`

- Get GETs os-server-groups/{id} with the raw ID, decodes the server_group envelope and accepts only 200.
  Tests: `compute/v2/servergroups/native_contracts_test.go:TestNativeServerGroupsRoutesBodiesAndDecode`, `compute/v2/servergroups/native_contracts_test.go:TestNativeServerGroupsStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native servergroups call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/servergroups.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.WithListQuery`

- List GETs os-server-groups with all_projects/limit/offset and extension query as one single page that ignores links, yields nothing for an empty array, surfaces io.EOF for a bodyless 204 and returns native pager status errors (200/204/300) without operation context.
  Tests: `compute/v2/servergroups/native_contracts_test.go:TestNativeServerGroupsRoutesBodiesAndDecode`, `compute/v2/servergroups/native_contracts_test.go:TestNativeServerGroupsStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native servergroups call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/servergroups.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.Delete`

- Delete DELETEs os-server-groups/{id} with the raw ID and accepts only 202/204.
  Tests: `compute/v2/servergroups/native_contracts_test.go:TestNativeServerGroupsRoutesBodiesAndDecode`, `compute/v2/servergroups/native_contracts_test.go:TestNativeServerGroupsStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native servergroups call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/servergroups/requests.go)
