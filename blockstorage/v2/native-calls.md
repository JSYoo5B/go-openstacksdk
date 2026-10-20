# Native Calls

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v2/volumes.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithCreateHintOpts`

- Create POSTs {"volume":{...}} to volumes (imageRef for ImageID) with extensions inside the envelope and non-empty scheduler hints as top-level OS-SCH-HNT:scheduler_hints, rejects non-UUID host/instance hints and core-key extension collisions before HTTP, decodes the volume key and accepts only 202. In v2 the size field has no omitempty, so zero is sent as "size":0 and the required tag does not reject it.
  Tests: `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Get`

- Get GETs volumes/{raw id}, decodes the volume key returning a zero volume for {} or null, wraps missing-key, non-object and zoned (trailing Z) timestamp decode errors, keeps bootable as a string, and accepts only 200. The v2 Volume has no backup_id or volume_image_metadata fields.
  Tests: `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithListQuery`

- List GETs volumes/detail with ListOpts (Python-dict-like metadata string, all_tenants) and extension query values, follows volumes_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP. v2 ListOpts has no bootable filter.
  Tests: `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithUpdateField`

- Update PUTs {"volume":{...}} to volumes/{id} sending explicit empty pointer name/description and non-empty metadata with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithDeleteQuery`

- Delete DELETEs volumes/{raw id} with cascade=true only when requested plus extension query values, rejects an empty query key before HTTP and accepts only 202/204.
  Tests: `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.WaitForStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.WaitForStatus`

- WaitForStatus polls Get immediately then on a one-second ticker until the status matches exactly (case-sensitive), returns Get errors at once and stops only on the caller context deadline, wrapping both in operation context.
  Tests: `blockstorage/v2/volumes/native_crud_contracts_test.go:TestNativeVolumeWaitForStatus`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.Attach`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Attach`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithAttachField`

- Attach POSTs to volumes/{id}/action {"os-attach":{...}} with extensions inside the action object, rejects a core-key extension collision before HTTP and accepts only 202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.BeginDetaching`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.BeginDetaching`

- BeginDetaching POSTs to volumes/{id}/action {"os-begin_detaching":{}} and accepts only 202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.Detach`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Detach`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithDetachField`

- Detach POSTs to volumes/{id}/action {"os-detach":{...}} (empty object without an attachment ID), rejects a nil option before HTTP and accepts only 202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.Reserve`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Reserve`

- Reserve POSTs to volumes/{id}/action {"os-reserve":{}} and accepts only 200/201/202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.Unreserve`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Unreserve`

- Unreserve POSTs to volumes/{id}/action {"os-unreserve":{}} and accepts only 200/201/202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.InitializeConnection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.InitializeConnection`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithInitializeConnectionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithInitializeConnectionField`

- InitializeConnection POSTs to volumes/{id}/action {"os-initialize_connection":{"connector":{...}}} with explicit false multipath and extensions beside connector, returns the raw connection_info map (nil without error when absent) and accepts only 200/201/202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.TerminateConnection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.TerminateConnection`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithTerminateConnectionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithTerminateConnectionField`

- TerminateConnection POSTs to volumes/{id}/action {"os-terminate_connection":{"connector":{...}}} and accepts only 202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.ExtendSize`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.ExtendSize`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithExtendSizeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithExtendSizeField`

- ExtendSize POSTs to volumes/{id}/action {"os-extend":{"new_size":N}} sending a zero size because the native required tag does not reject integer zero, and accepts only 202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.UploadImage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.UploadImage`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithUploadImageOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithUploadImageField`

- UploadImage POSTs to volumes/{id}/action {"os-volume_upload_image":{...}}, decodes VolumeImage with its volume type (zero value when the key is absent, error for zoned timestamps) and accepts only 202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.SetImageMetadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.SetImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithSetImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithSetImageMetadataField`

- SetImageMetadata POSTs to volumes/{id}/action {"os-set_image_metadata":{"metadata":...}} sending nil metadata as null and accepts only 200.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.SetBootable`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.SetBootable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithSetBootableOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithSetBootableField`

- SetBootable POSTs to volumes/{id}/action {"os-set_bootable":{"bootable":...}} sending false explicitly and accepts only 200.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.ChangeType`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.ChangeType`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithChangeTypeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithChangeTypeField`

- ChangeType POSTs to volumes/{id}/action {"os-retype":{"new_type":...,"migration_policy":...}}, rejects an empty new type before HTTP and accepts only 202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/volumes.ReImage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.ReImage`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithReImageOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithReImageField`

- ReImage POSTs to volumes/{id}/action {"os-reimage":{"image_id":...,"reimage_reserved":...}} always sending both fields and accepts only 202.
  Tests: `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v2/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes call only; Python volume Resource parity and SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/snapshots.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.WithCreateField`

- Create POSTs {"snapshot":{...}} to snapshots sending force only when true, requires volume_id and rejects a core-key extension collision before HTTP and accepts only 202.
  Tests: `blockstorage/v2/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesSinglePageAndDecode`, `blockstorage/v2/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/snapshots.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.Get`

- Get GETs snapshots/{raw id}, returns nil without error for a missing or null snapshot envelope, wraps non-object and zoned (trailing Z) timestamp decode errors and accepts only 200.
  Tests: `blockstorage/v2/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesSinglePageAndDecode`, `blockstorage/v2/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/snapshots.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.WithListQuery`

- List GETs the summary snapshots route with ListOpts (all_tenants included) and extension query values as a single page that never follows snapshots_links, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v2/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesSinglePageAndDecode`, `blockstorage/v2/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/snapshots.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.Delete`

- Delete DELETEs snapshots/{raw id} and accepts only 202/204.
  Tests: `blockstorage/v2/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesSinglePageAndDecode`, `blockstorage/v2/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v2/snapshots.WaitForStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.WaitForStatus`

- WaitForStatus polls Get immediately then on a one-second ticker until the status matches exactly (case-sensitive), returns Get errors at once and stops on the caller context deadline with operation context; a poll returning no snapshot envelope returns an OperationError wrapping resource.ErrUnsupported instead of the native nil dereference.
  Tests: `blockstorage/v2/snapshots/native_contracts_test.go:TestNativeSnapshotWaitForStatus`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.
- The generated method calls an SDK helper with the native request, interval and status comparison, adding only a nil-snapshot guard where native WaitForStatus panics.

### `gophercloud:blockstorage/v2/backups.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithCreateField`

- Create POSTs {"backup":{...}} to backups sending incremental/force only when true, requires volume_id and rejects a core-key extension collision before HTTP, decodes project, pointer metadata/availability zone and data timestamp, and accepts only 202.
  Tests: `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/backups.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Get`

- Get GETs backups/{raw id}, decodes the backup key returning a zero backup (nil metadata) for {} or null, wraps missing-key and zoned timestamp decode errors and accepts only 200.
  Tests: `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/backups.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithListQuery`

- List GETs the summary backups route with ListOpts (all_tenants included) and extension query values, follows backups_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/backups.ListDetail`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithListDetailOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithListDetailQuery`

- ListDetail GETs backups/detail with the separate ListDetailOpts (with_count, sort, all_tenants) and extension query values, follows backups_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/backups.RestoreFromBackup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.RestoreFromBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithRestoreFromBackupOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithRestoreFromBackupField`

- RestoreFromBackup POSTs {"restore":{...}} (empty object for empty options) to backups/{id}/restore with extensions inside it, rejects a nil option before HTTP, decodes the restore key and accepts only 202.
  Tests: `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/backups.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Delete`

- Delete DELETEs backups/{raw id} and accepts only 202/204.
  Tests: `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v2/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v2/backups.Update`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithUpdateField`

- The v2 facade forwards the native flat name/description/metadata body to PUT backups/{id} without the SDK backup-envelope repair applied to v3; the generator test pins that only the v3 package receives the repair.
  Tests: `internal/cmd/sdkgen/backup_update_body_test.go:TestBackupUpdateBodyRepairHasExactPackageAndBuilderScope`.

Python/Go differences:

- Cinder added backup update in the v3 API at microversion 3.9, so the v2 route is not usable and the native body lacks the backup envelope.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

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

### `gophercloud:blockstorage/v2/volumes.ForceDelete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.ForceDelete`

- ForceDelete POSTs {"os-force_delete":""} (an empty string) to volumes/{id}/action and accepts the native POST default 201/202.
  Tests: `blockstorage/v2/volumes/native_admin_contracts_test.go:TestNativeVolumeAdminActionRoutesBodiesAndStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes admin action only; the default Cinder policy limits it to administrators, and SDK-owned reset-status workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and bodies are unchanged.

### `gophercloud:blockstorage/v2/volumes.ResetStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.ResetStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithResetStatusOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithResetStatusField`

- ResetStatus POSTs {"os-reset_status":{"status":...}} to volumes/{id}/action always sending status (empty included) and omitting empty migration/attach status, keeps extensions inside the action object, rejects a status extension collision before HTTP and accepts only 202.
  Tests: `blockstorage/v2/volumes/native_admin_contracts_test.go:TestNativeVolumeAdminActionRoutesBodiesAndStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/volumes admin action only; the default Cinder policy limits it to administrators, and SDK-owned reset-status workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and bodies are unchanged.

### `gophercloud:blockstorage/v2/backups.Export`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Export`

- Export GETs backups/{id}/export_record, decodes the backup-record key with backup_url base64-decoded into bytes (empty record when the key is absent, wrapped error for invalid base64) and accepts only 200.
  Tests: `blockstorage/v2/backups/native_admin_contracts_test.go:TestNativeBackupAdminCalls`, `blockstorage/v2/backups/native_admin_contracts_test.go:TestNativeBackupAdminStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups admin call only; the default Cinder policy limits it to administrators, and the SDK-owned ExportRecord, ImportBackup and reset workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, base64 record handling and decoding are unchanged.

### `gophercloud:blockstorage/v2/backups.Import`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Import`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithImportOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithImportField`

- Import POSTs {"backup-record":{...}} to backups/import_record re-encoding backup_url bytes as base64 with extensions inside the record, rejects a core-key collision before HTTP, decodes the backup ID and name and accepts only 201.
  Tests: `blockstorage/v2/backups/native_admin_contracts_test.go:TestNativeBackupAdminCalls`, `blockstorage/v2/backups/native_admin_contracts_test.go:TestNativeBackupAdminStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups admin call only; the default Cinder policy limits it to administrators, and the SDK-owned ExportRecord, ImportBackup and reset workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, base64 record handling and decoding are unchanged.

### `gophercloud:blockstorage/v2/backups.ResetStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.ResetStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithResetStatusOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithResetStatusField`

- ResetStatus POSTs {"os-reset_status":{"status":...}} to backups/{id}/action, rejects a nil option before HTTP and accepts only 202.
  Tests: `blockstorage/v2/backups/native_admin_contracts_test.go:TestNativeBackupAdminCalls`, `blockstorage/v2/backups/native_admin_contracts_test.go:TestNativeBackupAdminStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups admin call only; the default Cinder policy limits it to administrators, and the SDK-owned ExportRecord, ImportBackup and reset workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, base64 record handling and decoding are unchanged.

### `gophercloud:blockstorage/v2/backups.ForceDelete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.ForceDelete`

- ForceDelete POSTs {"os-force_delete":{}} to backups/{id}/action and accepts only 202.
  Tests: `blockstorage/v2/backups/native_admin_contracts_test.go:TestNativeBackupAdminCalls`, `blockstorage/v2/backups/native_admin_contracts_test.go:TestNativeBackupAdminStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v2/backups admin call only; the default Cinder policy limits it to administrators, and the SDK-owned ExportRecord, ImportBackup and reset workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, base64 record handling and decoding are unchanged.

## References

- [Native Volumes](../v3/native-volumes.md)
- [Native Snapshots](../v3/native-snapshots.md)
- [Native Backups](../v3/native-backups.md)
- [Native Transfers](../v3/native-transfers.md)
