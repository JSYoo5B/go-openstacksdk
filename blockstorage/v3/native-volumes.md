# Native Volumes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/volumes.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCreateHintOpts`

- Create POSTs {"volume":{...}} to volumes (imageRef for ImageID) with extensions inside the envelope and non-empty scheduler hints as top-level OS-SCH-HNT:scheduler_hints, rejects non-UUID host/instance hints and core-key extension collisions before HTTP, decodes the volume key and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes call only; Python volume Resource parity and the SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Get`

- Get GETs volumes/{raw id}, decodes the volume key returning a zero volume for {} or null, wraps missing-key, non-object and zoned (trailing Z) timestamp decode errors, keeps bootable as a string and null backup_id as nil, and accepts only 200.
  Tests: `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes call only; Python volume Resource parity and the SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithListQuery`

- List GETs volumes/detail with ListOpts (Python-dict-like metadata string, all_tenants, pointer bootable including false) and extension query values, follows volumes_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes call only; Python volume Resource parity and the SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithUpdateField`

- Update PUTs {"volume":{...}} to volumes/{id} sending explicit empty pointer name/description and non-empty metadata with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes call only; Python volume Resource parity and the SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithDeleteQuery`

- Delete DELETEs volumes/{raw id} with cascade=true only when requested plus extension query values, rejects an empty query key before HTTP and accepts only 202/204.
  Tests: `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeRoutesBodiesPagingAndDecode`, `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes call only; Python volume Resource parity and the SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.WaitForStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.WaitForStatus`

- WaitForStatus polls Get immediately then on a one-second ticker until the status matches exactly (case-sensitive), returns Get errors at once and stops only on the caller context deadline, wrapping both in operation context.
  Tests: `blockstorage/v3/volumes/native_crud_contracts_test.go:TestNativeVolumeWaitForStatus`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes call only; Python volume Resource parity and the SDK-owned volume workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.Attach`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Attach`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithAttachField`

- Attach POSTs to volumes/{id}/action {"os-attach":{...}} with extensions inside the action object, rejects a core-key extension collision before HTTP and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.BeginDetaching`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.BeginDetaching`

- BeginDetaching POSTs to volumes/{id}/action {"os-begin_detaching":{}} and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.Detach`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Detach`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithDetachField`

- Detach POSTs to volumes/{id}/action {"os-detach":{...}} (empty object without an attachment ID), rejects a nil option before HTTP and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.Reserve`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Reserve`

- Reserve POSTs to volumes/{id}/action {"os-reserve":{}} and accepts only 200/201/202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.Unreserve`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Unreserve`

- Unreserve POSTs to volumes/{id}/action {"os-unreserve":{}} and accepts only 200/201/202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.InitializeConnection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.InitializeConnection`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithInitializeConnectionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithInitializeConnectionField`

- InitializeConnection POSTs to volumes/{id}/action {"os-initialize_connection":{"connector":{...}}} with explicit false multipath and extensions beside connector, returns the raw connection_info map (nil without error when absent) and accepts only 200/201/202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.TerminateConnection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.TerminateConnection`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithTerminateConnectionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithTerminateConnectionField`

- TerminateConnection POSTs to volumes/{id}/action {"os-terminate_connection":{"connector":{...}}} and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.ExtendSize`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ExtendSize`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithExtendSizeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithExtendSizeField`

- ExtendSize POSTs to volumes/{id}/action {"os-extend":{"new_size":N}} sending a zero size because the native required tag does not reject integer zero, and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.UploadImage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.UploadImage`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithUploadImageOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithUploadImageField`

- UploadImage POSTs to volumes/{id}/action {"os-volume_upload_image":{...}}, decodes VolumeImage with its volume type (zero value when the key is absent, error for zoned timestamps) and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.SetImageMetadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SetImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSetImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSetImageMetadataField`

- SetImageMetadata POSTs to volumes/{id}/action {"os-set_image_metadata":{"metadata":...}} sending nil metadata as null and accepts only 200.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.SetBootable`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SetBootable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSetBootableOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSetBootableField`

- SetBootable POSTs to volumes/{id}/action {"os-set_bootable":{"bootable":...}} sending false explicitly and accepts only 200.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.ChangeType`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ChangeType`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithChangeTypeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithChangeTypeField`

- ChangeType POSTs to volumes/{id}/action {"os-retype":{"new_type":...,"migration_policy":...}}, rejects an empty new type before HTTP and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/volumes.ReImage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ReImage`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithReImageOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithReImageField`

- ReImage POSTs to volumes/{id}/action {"os-reimage":{"image_id":...,"reimage_reserved":...}} always sending both fields and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionRoutesAndBodies`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionStrictStatuses`, `blockstorage/v3/volumes/native_action_contracts_test.go:TestNativeVolumeActionPreflightAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes action call only; Python volume Resource actions and the SDK-owned attach, resize, image and flag workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

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

### `gophercloud:blockstorage/v3/volumes.ForceDelete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ForceDelete`

- ForceDelete POSTs {"os-force_delete":""} (an empty string) to volumes/{id}/action and accepts the native POST default 201/202.
  Tests: `blockstorage/v3/volumes/native_admin_contracts_test.go:TestNativeVolumeAdminActionRoutesBodiesAndStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes admin action only; the default Cinder policy limits it to administrators, and SDK-owned reset-status workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and bodies are unchanged.

### `gophercloud:blockstorage/v3/volumes.ResetStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ResetStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithResetStatusOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithResetStatusField`

- ResetStatus POSTs {"os-reset_status":{"status":...}} to volumes/{id}/action always sending status (empty included) and omitting empty migration/attach status, keeps extensions inside the action object, rejects a status extension collision before HTTP and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_admin_contracts_test.go:TestNativeVolumeAdminActionRoutesBodiesAndStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes admin action only; the default Cinder policy limits it to administrators, and SDK-owned reset-status workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and bodies are unchanged.

### `gophercloud:blockstorage/v3/volumes.Unmanage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Unmanage`

- Unmanage POSTs {"os-unmanage":{}} to volumes/{id}/action and accepts only 202.
  Tests: `blockstorage/v3/volumes/native_admin_contracts_test.go:TestNativeVolumeAdminActionRoutesBodiesAndStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/volumes admin action only; the default Cinder policy limits it to administrators, and SDK-owned reset-status workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and bodies are unchanged.

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

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/requests.go)
- [..](../README.md)
- [Service Waits](../../docs/service-waits.md)
