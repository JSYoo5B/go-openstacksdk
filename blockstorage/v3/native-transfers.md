# Native Transfers

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/transfers.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithCreateField`

- Create POSTs {"transfer":{...}} to os-volume-transfer, requires volume_id and rejects a core-key extension collision before HTTP, decodes the transfer key including the auth_key secret and links, and accepts only 202.
  Tests: `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/transfers.Accept`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Accept`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithAcceptOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithAcceptField`

- Accept POSTs {"accept":{"auth_key":...}} to os-volume-transfer/{id}/accept with extensions inside the accept object, requires auth_key and rejects a nil option before HTTP, decodes the transfer key and accepts only 202.
  Tests: `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/transfers.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Get`

- Get GETs os-volume-transfer/{raw id}, decodes the transfer key returning a zero value for {} or null, wraps missing-key and zoned timestamp decode errors and accepts only 200.
  Tests: `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/transfers.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithListQuery`

- List GETs os-volume-transfer/detail with ListOpts (all_tenants included) and extension query values, follows transfers_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/transfers.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Delete`

- Delete DELETEs os-volume-transfer/{raw id} and accepts only 202/204.
  Tests: `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v3/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/availabilityzones.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/availabilityzones.API.List`

- List GETs os-availability-zone once and extracts the availabilityZoneInfo array without the inherited array-only IsEmpty check or link following, yields nothing for an empty or missing list, reports extraction errors for a non-array list or bare array body, and native status errors.
  Tests: `blockstorage/v3/availabilityzones/native_contracts_test.go:TestNativeVolumeAvailabilityZonesSinglePageObjectEnvelope`, `blockstorage/v3/availabilityzones/native_contracts_test.go:TestNativeVolumeAvailabilityZonesStatusesAndDecode`, `internal/cmd/sdkgen/page_extractors_test.go:TestPinnedForeignPagesAndLocalWrapperGenerateCompatibleIterators`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/availabilityzones call only; Python availability zone Resource parity is tracked separately.
- The page embeds SinglePageBase without overriding its array-only IsEmpty, so the generated facade reads it like native AllPages (resource.SinglePageStream) instead of page-by-page iteration, which would reject every object response. URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/apiversions.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/apiversions.API.List`

- List GETs the client Endpoint root with the version segment and query removed (ignoring ResourceBase), accepts the native 200/204/300 page statuses including Cinder's 300 Multiple Choices, decodes RFC3339 updated times (malformed times fail extraction), and reports io.EOF for a bodyless 204.
  Tests: `blockstorage/apiversions/native_contracts_test.go:TestNativeVolumeAPIVersionsUseTheUnversionedRoot`, `blockstorage/apiversions/native_contracts_test.go:TestNativeVolumeAPIVersionsStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/apiversions call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/transfers.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.WithCreateField`

- Create POSTs {"transfer":{...}} to os-volume-transfer, requires volume_id and rejects a core-key extension collision before HTTP, decodes the transfer key including the auth_key secret and links, and accepts only 202.
  Tests: `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/transfers.Accept`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.Accept`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.WithAcceptOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.WithAcceptField`

- Accept POSTs {"accept":{"auth_key":...}} to os-volume-transfer/{id}/accept with extensions inside the accept object, requires auth_key and rejects a nil option before HTTP, decodes the transfer key and accepts only 202.
  Tests: `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/transfers.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.Get`

- Get GETs os-volume-transfer/{raw id}, decodes the transfer key returning a zero value for {} or null, wraps missing-key and zoned timestamp decode errors and accepts only 200.
  Tests: `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/transfers.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.WithListQuery`

- List GETs os-volume-transfer/detail with ListOpts (all_tenants included) and extension query values, follows transfers_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/transfers.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.Delete`

- Delete DELETEs os-volume-transfer/{raw id} and accepts only 202/204.
  Tests: `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferRoutesBodiesPagingAndDecode`, `blockstorage/v2/transfers/native_contracts_test.go:TestNativeTransferStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/transfers call only; Python Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/availabilityzones.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/availabilityzones.API.List`

- List GETs os-availability-zone once and extracts the availabilityZoneInfo array without the inherited array-only IsEmpty check or link following, yields nothing for an empty or missing list, reports extraction errors for a non-array list or bare array body, and native status errors.
  Tests: `blockstorage/v2/availabilityzones/native_contracts_test.go:TestNativeVolumeAvailabilityZonesSinglePageObjectEnvelope`, `blockstorage/v2/availabilityzones/native_contracts_test.go:TestNativeVolumeAvailabilityZonesStatusesAndDecode`, `internal/cmd/sdkgen/page_extractors_test.go:TestPinnedForeignPagesAndLocalWrapperGenerateCompatibleIterators`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/availabilityzones call only; Python availability zone Resource parity is tracked separately.
- The page embeds SinglePageBase without overriding its array-only IsEmpty, so the generated facade reads it like native AllPages (resource.SinglePageStream) instead of page-by-page iteration, which would reject every object response. URL construction, OkCodes and decoding are unchanged.

## References

- [v3 transfers](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/transfers/requests.go)
- [v3 availability zones](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/availabilityzones/requests.go)
- [API versions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/apiversions/requests.go)
