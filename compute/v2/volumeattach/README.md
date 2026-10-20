# Volumeattach

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/volumeattach.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.WithCreateField`

- Create POSTs {"volumeAttachment":{...}} to servers/{id}/os-volume_attachments omitting empty fields, places extensions inside the envelope, rejects a missing volumeId, a core-key extension collision and a nil option before HTTP, decodes the response and accepts only 200.
  Tests: `compute/v2/volumeattach/native_contracts_test.go:TestNativeVolumeAttachRoutesBodiesAndDecode`, `compute/v2/volumeattach/native_contracts_test.go:TestNativeVolumeAttachStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native volumeattach call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/volumeattach.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.API.Get`

- Get GETs servers/{id}/os-volume_attachments/{volumeID} with the raw ID, decodes the volumeAttachment envelope and accepts only 200.
  Tests: `compute/v2/volumeattach/native_contracts_test.go:TestNativeVolumeAttachRoutesBodiesAndDecode`, `compute/v2/volumeattach/native_contracts_test.go:TestNativeVolumeAttachStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native volumeattach call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/volumeattach.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.API.List`

- List GETs servers/{id}/os-volume_attachments as one single page that ignores links, yields nothing for an empty array, surfaces io.EOF for a bodyless 204 and returns native pager status errors (200/204/300) without operation context.
  Tests: `compute/v2/volumeattach/native_contracts_test.go:TestNativeVolumeAttachRoutesBodiesAndDecode`, `compute/v2/volumeattach/native_contracts_test.go:TestNativeVolumeAttachStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native volumeattach call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/volumeattach.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach.API.Delete`

- Delete DELETEs servers/{id}/os-volume_attachments/{volumeID} with the raw ID and accepts only 202/204.
  Tests: `compute/v2/volumeattach/native_contracts_test.go:TestNativeVolumeAttachRoutesBodiesAndDecode`, `compute/v2/volumeattach/native_contracts_test.go:TestNativeVolumeAttachStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native volumeattach call only; Python compute resource parity and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/volumeattach/requests.go)
