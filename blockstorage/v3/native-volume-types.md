# Native Volume Types

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/volumetypes.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.Get`

- Get GETs types/{raw id}, decodes the volume_type key (zero value for {} or null) including is_public, os-volume-type-access:is_public and extra specs, and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_read_contracts_test.go:TestNativeVolumeTypeReadRoutesPagingAndDecode`, `blockstorage/v3/volumetypes/native_read_contracts_test.go:TestNativeVolumeTypeReadStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes read call only; Python volume type Resource parity and admin type management are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithListQuery`

- List GETs types always sending is_public=None when visibility is unset (true/false otherwise) with ListOpts and extension query values, follows only the singular volume_type_links next href (not volume_types_links), yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/volumetypes/native_read_contracts_test.go:TestNativeVolumeTypeReadRoutesPagingAndDecode`, `blockstorage/v3/volumetypes/native_read_contracts_test.go:TestNativeVolumeTypeReadStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes read call only; Python volume type Resource parity and admin type management are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.ListExtraSpecs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.ListExtraSpecs`

- ListExtraSpecs GETs types/{id}/extra_specs, returns the extra_specs string map (nil when absent) and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_read_contracts_test.go:TestNativeVolumeTypeReadRoutesPagingAndDecode`, `blockstorage/v3/volumetypes/native_read_contracts_test.go:TestNativeVolumeTypeReadStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes read call only; Python volume type Resource parity and admin type management are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumetypes.GetExtraSpec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.GetExtraSpec`

- GetExtraSpec GETs types/{id}/extra_specs/{unescaped key}, returns the envelope-free body as a string map, wraps a decode error for non-string values and accepts only 200.
  Tests: `blockstorage/v3/volumetypes/native_read_contracts_test.go:TestNativeVolumeTypeReadRoutesPagingAndDecode`, `blockstorage/v3/volumetypes/native_read_contracts_test.go:TestNativeVolumeTypeReadStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumetypes read call only; Python volume type Resource parity and admin type management are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumetypes/requests.go)
