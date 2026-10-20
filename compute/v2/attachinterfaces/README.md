# Attachinterfaces

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/attachinterfaces.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.WithCreateField`

- Create POSTs {"interfaceAttachment":{...}} to servers/{id}/os-interface omitting empty fields, places extensions inside the envelope, rejects a core-key extension collision and a nil option before HTTP, decodes the response and accepts only 200.
  Tests: `compute/v2/attachinterfaces/native_contracts_test.go:TestNativeAttachInterfacesRoutesBodiesAndDecode`, `compute/v2/attachinterfaces/native_contracts_test.go:TestNativeAttachInterfacesStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native attachinterfaces call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/attachinterfaces.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.API.Get`

- Get GETs servers/{id}/os-interface/{portID} with the raw ID, decodes the interfaceAttachment envelope and accepts only 200.
  Tests: `compute/v2/attachinterfaces/native_contracts_test.go:TestNativeAttachInterfacesRoutesBodiesAndDecode`, `compute/v2/attachinterfaces/native_contracts_test.go:TestNativeAttachInterfacesStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native attachinterfaces call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/attachinterfaces.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.API.List`

- List GETs servers/{id}/os-interface as one single page that ignores links, yields nothing for an empty array, surfaces io.EOF for a bodyless 204 and returns native pager status errors (200/204/300) without operation context.
  Tests: `compute/v2/attachinterfaces/native_contracts_test.go:TestNativeAttachInterfacesRoutesBodiesAndDecode`, `compute/v2/attachinterfaces/native_contracts_test.go:TestNativeAttachInterfacesStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native attachinterfaces call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/attachinterfaces.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces.API.Delete`

- Delete DELETEs servers/{id}/os-interface/{portID} with the raw ID and accepts only 202/204.
  Tests: `compute/v2/attachinterfaces/native_contracts_test.go:TestNativeAttachInterfacesRoutesBodiesAndDecode`, `compute/v2/attachinterfaces/native_contracts_test.go:TestNativeAttachInterfacesStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native attachinterfaces call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/attachinterfaces/requests.go)
