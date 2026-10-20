# Native Attachments

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/attachments.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithCreateField`

- Create POSTs {"attachment":{...}} to attachments with the caller microversion header, always sending volume_uuid and instance_uuid (empty included) with extensions inside the envelope, rejects a core-key extension collision before HTTP, decodes the attachment key and accepts only 200/202.
  Tests: `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentRoutesBodiesPagingAndDecode`, `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/attachments call only; the SDK-owned Cinder attachment workflow and Python Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling, decoding and caller-selected microversion headers are unchanged.

### `gophercloud:blockstorage/v3/attachments.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Get`

- Get GETs attachments/{raw id}, decodes the attachment key returning a zero value for {} or null, wraps missing-key and zoned timestamp decode errors, keeps connection_info raw and accepts only 200.
  Tests: `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentRoutesBodiesPagingAndDecode`, `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/attachments call only; the SDK-owned Cinder attachment workflow and Python Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling, decoding and caller-selected microversion headers are unchanged.

### `gophercloud:blockstorage/v3/attachments.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithListQuery`

- List GETs attachments/detail with ListOpts (all_tenants included) and extension query values, ignores the Cinder attachments_links array and follows only a links.next string because the page defines no NextPageURL, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentRoutesBodiesPagingAndDecode`, `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/attachments call only; the SDK-owned Cinder attachment workflow and Python Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling, decoding and caller-selected microversion headers are unchanged.

### `gophercloud:blockstorage/v3/attachments.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithUpdateField`

- Update PUTs {"attachment":{"connector":...}} to attachments/{id} sending a nil connector as null with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentRoutesBodiesPagingAndDecode`, `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/attachments call only; the SDK-owned Cinder attachment workflow and Python Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling, decoding and caller-selected microversion headers are unchanged.

### `gophercloud:blockstorage/v3/attachments.Complete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Complete`

- Complete POSTs {"os-complete":null} to attachments/{id}/action and accepts only 204.
  Tests: `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentRoutesBodiesPagingAndDecode`, `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/attachments call only; the SDK-owned Cinder attachment workflow and Python Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling, decoding and caller-selected microversion headers are unchanged.

### `gophercloud:blockstorage/v3/attachments.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Delete`

- Delete DELETEs attachments/{raw id} and accepts only 200, rejecting 202 and 204.
  Tests: `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentRoutesBodiesPagingAndDecode`, `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/attachments call only; the SDK-owned Cinder attachment workflow and Python Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling, decoding and caller-selected microversion headers are unchanged.

### `gophercloud:blockstorage/v3/attachments.WaitForStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.WaitForStatus`

- WaitForStatus polls Get immediately then on a one-second ticker until the status matches exactly, treats a missing envelope as an empty status and keeps polling until the caller deadline, and returns Get errors at once with operation context.
  Tests: `blockstorage/v3/attachments/native_contracts_test.go:TestNativeAttachmentWaitForStatus`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/attachments call only; the SDK-owned Cinder attachment workflow and Python Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling, decoding and caller-selected microversion headers are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/attachments/requests.go)
- [Cinder Volume Attachment](../cinder-volume-attachment.md)
