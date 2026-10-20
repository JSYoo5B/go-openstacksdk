# Native Snapshots

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/snapshots.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithCreateField`

- Create POSTs {"snapshot":{...}} to snapshots sending force only when true, requires volume_id and rejects a core-key extension collision before HTTP, decodes extended progress/project attributes and accepts only 202.
  Tests: `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesPagingAndDecode`, `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/snapshots.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.Get`

- Get GETs snapshots/{raw id}, returns nil without error for a missing or null snapshot envelope, wraps non-object and zoned (trailing Z) timestamp decode errors and accepts only 200.
  Tests: `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesPagingAndDecode`, `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/snapshots.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithListQuery`

- List GETs the summary snapshots route with ListOpts (all_tenants included) and extension query values, follows snapshots_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesPagingAndDecode`, `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/snapshots.ListDetail`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithListDetailOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithListDetailQuery`

- ListDetail GETs snapshots/detail with ListOpts (all_tenants included) and extension query values, follows snapshots_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesPagingAndDecode`, `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/snapshots.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithUpdateField`

- Update PUTs {"snapshot":{...}} to snapshots/{id} sending explicit empty pointer name/description with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesPagingAndDecode`, `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/snapshots.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.Delete`

- Delete DELETEs snapshots/{raw id} and accepts only 202/204.
  Tests: `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotRoutesBodiesPagingAndDecode`, `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.

### `gophercloud:blockstorage/v3/snapshots.WaitForStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.WaitForStatus`

- WaitForStatus polls Get immediately then on a one-second ticker until the status matches exactly (case-sensitive), returns Get errors at once and stops on the caller context deadline with operation context; a poll returning no snapshot envelope returns an OperationError wrapping resource.ErrUnsupported instead of the native nil dereference.
  Tests: `blockstorage/v3/snapshots/native_contracts_test.go:TestNativeSnapshotWaitForStatus`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots call only; Python snapshot Resource parity, metadata scopes and the SDK-owned waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging, polling and decoding are unchanged.
- The generated method calls an SDK helper with the native request, interval and status comparison, adding only a nil-snapshot guard where native WaitForStatus panics.

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

### `gophercloud:blockstorage/v3/snapshots.ForceDelete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.ForceDelete`

- ForceDelete POSTs {"os-force_delete":{}} to snapshots/{id}/action and accepts only 202.
  Tests: `blockstorage/v3/snapshots/native_admin_contracts_test.go:TestNativeSnapshotAdminActions`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots admin action only; the default Cinder policy limits it to administrators, and SDK-owned reset-status workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and bodies are unchanged.

### `gophercloud:blockstorage/v3/snapshots.ResetStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.ResetStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithResetStatusOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithResetStatusField`

- ResetStatus POSTs {"os-reset_status":{"status":...}} to snapshots/{id}/action with extensions inside the action object, rejects a status extension collision before HTTP and accepts only 202.
  Tests: `blockstorage/v3/snapshots/native_admin_contracts_test.go:TestNativeSnapshotAdminActions`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots admin action only; the default Cinder policy limits it to administrators, and SDK-owned reset-status workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and bodies are unchanged.

### `gophercloud:blockstorage/v3/snapshots.UpdateStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.UpdateStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithUpdateStatusOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithUpdateStatusField`

- UpdateStatus POSTs {"os-update_snapshot_status":{"status":...,"progress":...}} to snapshots/{id}/action always sending status (empty included) and omitting empty progress, rejects a nil option before HTTP and accepts only 202.
  Tests: `blockstorage/v3/snapshots/native_admin_contracts_test.go:TestNativeSnapshotAdminActions`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/snapshots admin action only; the default Cinder policy limits it to administrators, and SDK-owned reset-status workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and bodies are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/snapshots/requests.go)
- [Snapshot metadata](snapshots/README.md)
