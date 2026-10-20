# Native Backups

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/backups.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithCreateField`

- Create POSTs {"backup":{...}} to backups sending incremental/force only when true, requires volume_id and rejects a core-key extension collision before HTTP, decodes project, pointer metadata/availability zone and data timestamp, and accepts only 202.
  Tests: `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/backups.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Get`

- Get GETs backups/{raw id}, decodes the backup key returning a zero backup (nil metadata) for {} or null, wraps missing-key and zoned timestamp decode errors and accepts only 200.
  Tests: `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/backups.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithListQuery`

- List GETs the summary backups route with ListOpts (all_tenants included) and extension query values, follows backups_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/backups.ListDetail`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithListDetailOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithListDetailQuery`

- ListDetail GETs backups/detail with the separate ListDetailOpts (with_count, sort, all_tenants) and extension query values, follows backups_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/backups.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithUpdateField`

- Update PUTs to backups/{id} with the SDK adapter moving native flat name/description/metadata into {"backup":{...}} together with extension fields, keeps nil omission and explicit empty metadata, and accepts only 200.
  Tests: `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`, `blockstorage/v3/backups/update_contract_test.go:TestBackupUpdateTypedFieldsKeepNilOmissionAndExplicitEmptyMetadata`, `blockstorage/v3/backups/update_contract_test.go:TestBackupUpdateExtensionsStayInsideOwnedBackupEnvelopeAndProtectCoreInputs`, `blockstorage/v3/backups/update_contract_test.go:TestBackupUpdateExtractsActualSummaryAndKeepsNativeAndOptionFailures`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups Update call; Python backup Resource parity is tracked separately.
- Native Update sends the fields without the backup envelope Cinder requires; the generated SDK adapter adds that envelope and keeps extensions inside it. URL construction, OkCodes and decoding are unchanged.

### `gophercloud:blockstorage/v3/backups.RestoreFromBackup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.RestoreFromBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithRestoreFromBackupOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithRestoreFromBackupField`

- RestoreFromBackup POSTs {"restore":{...}} (empty object for empty options) to backups/{id}/restore with extensions inside it, rejects a nil option before HTTP, decodes the restore key and accepts only 202.
  Tests: `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:blockstorage/v3/backups.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Delete`

- Delete DELETEs backups/{raw id} and accepts only 202/204.
  Tests: `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupRoutesBodiesPagingAndDecode`, `blockstorage/v3/backups/native_crud_contracts_test.go:TestNativeBackupStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups call only; Python backup Resource parity and the SDK-owned export, import, restore and status workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

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

### `gophercloud:blockstorage/v3/backups.Export`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Export`

- Export GETs backups/{id}/export_record, decodes the backup-record key with backup_url base64-decoded into bytes (empty record when the key is absent, wrapped error for invalid base64) and accepts only 200.
  Tests: `blockstorage/v3/backups/native_admin_contracts_test.go:TestNativeBackupAdminCalls`, `blockstorage/v3/backups/native_admin_contracts_test.go:TestNativeBackupAdminStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups admin call only; the default Cinder policy limits it to administrators, and the SDK-owned ExportRecord, ImportBackup and reset workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, base64 record handling and decoding are unchanged.

### `gophercloud:blockstorage/v3/backups.Import`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Import`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithImportOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithImportField`

- Import POSTs {"backup-record":{...}} to backups/import_record re-encoding backup_url bytes as base64 with extensions inside the record, rejects a core-key collision before HTTP, decodes the backup ID and name and accepts only 201.
  Tests: `blockstorage/v3/backups/native_admin_contracts_test.go:TestNativeBackupAdminCalls`, `blockstorage/v3/backups/native_admin_contracts_test.go:TestNativeBackupAdminStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups admin call only; the default Cinder policy limits it to administrators, and the SDK-owned ExportRecord, ImportBackup and reset workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, base64 record handling and decoding are unchanged.

### `gophercloud:blockstorage/v3/backups.ResetStatus`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ResetStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithResetStatusOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithResetStatusField`

- ResetStatus POSTs {"os-reset_status":{"status":...}} to backups/{id}/action, rejects a nil option before HTTP and accepts only 202.
  Tests: `blockstorage/v3/backups/native_admin_contracts_test.go:TestNativeBackupAdminCalls`, `blockstorage/v3/backups/native_admin_contracts_test.go:TestNativeBackupAdminStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups admin call only; the default Cinder policy limits it to administrators, and the SDK-owned ExportRecord, ImportBackup and reset workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, base64 record handling and decoding are unchanged.

### `gophercloud:blockstorage/v3/backups.ForceDelete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ForceDelete`

- ForceDelete POSTs {"os-force_delete":{}} to backups/{id}/action and accepts only 202.
  Tests: `blockstorage/v3/backups/native_admin_contracts_test.go:TestNativeBackupAdminCalls`, `blockstorage/v3/backups/native_admin_contracts_test.go:TestNativeBackupAdminStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/v3/backups admin call only; the default Cinder policy limits it to administrators, and the SDK-owned ExportRecord, ImportBackup and reset workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, base64 record handling and decoding are unchanged.

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

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go)
- [Backups](backups/README.md)
