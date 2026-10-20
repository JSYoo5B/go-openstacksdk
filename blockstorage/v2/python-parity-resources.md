# Python Parity Resources

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v2/backups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithListDetailQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithListDetailOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithListOptions`

- ListDetail (details=True) and List (details=False) send backups/detail and backups with the name, status, volume_id, all_tenants=True, sort_key, sort_dir and limit queries and follow backups_links.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupListDetailsAndQuery`.

Python/Go differences:

- ListDetailOpts has no typed name, status, volume_id, project_id, sort_key or sort_dir filters; WithListDetailQuery adds them.
- Python all_tenants=True sends all_tenants=True; typed Go AllTenants sends all_tenants=true, so WithListDetailQuery("all_tenants", "True") is needed for the identical string.
- Python filters returned rows locally for Body attributes outside the query mapping; Go sends only query parameters and leaves local filtering to the caller.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversion: neither Python v2 Resources nor the Go v2 client send a volume version header unless a session default or client Microversion is configured.

### `python:block_storage/v2/get_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Get`

- Get sends GET backups/{id} and decodes name, status, volume_id, object_count, is_incremental, container and availability_zone.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupGetCreateRestoreExportImportAndReset`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversion: neither Python v2 Resources nor the Go v2 client send a volume version header unless a session default or client Microversion is configured.

### `python:block_storage/v2/find_backup`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Find`

- Find(resource.ID) returns the GET 403 instead of falling back, and Find(resource.Name) lists backups/detail without a name query and matches locally.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupFindHasNoProxyFallback`.

Python/Go differences:

- Python matches the list by ID or name; Go Find(resource.Name) matches the name only.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/create_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.WithCreateField`

- Create posts the backup envelope with incremental, name and volume_id like Backup._transform_create_request renames is_incremental.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupGetCreateRestoreExportImportAndReset`.

Python/Go differences:

- Go CreateOpts omits false and empty values that Python would send when passed explicitly; WithCreateField can send them.
- Python drops attributes not declared on the v2 Backup resource, such as metadata; Go sends CreateOpts.Metadata when set.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Go accepts only 202.

### `python:block_storage/v2/delete_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.ForceDelete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE backups/{id}, treats 202 as success, ignores 404 by default, returns resource.ErrNotFound with WithMissingError and keeps 409.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupDeleteIgnoresMissing`.
- ForceDelete posts os-force_delete to backups/{id}/action for force=True.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupGetCreateRestoreExportImportAndReset`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- force=True posts {"os-force_delete": null} in Python and does not apply ignore_missing; Go ForceDelete posts {"os-force_delete": {}}.

### `python:block_storage/v2/import_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Import`

- Import posts backup-record with backup_service and the base64 backup_url to backups/import_record and returns the imported id and name.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupGetCreateRestoreExportImportAndReset`.

Python/Go differences:

- The pinned Backup.import_record overwrites the url argument with the backups/export_record path and posts there; Go Import repairs the route to backups/import_record and sends the caller locator.
- ImportOpts.BackupURL is bytes that encoding/json writes as standard base64, so the caller passes the decoded export locator rather than an arbitrary string.
- Python returns a Backup resource; Go returns ImportResponse with only ID and Name. Go accepts only 201.

### `python:block_storage/v2/export_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Export`

- Export sends GET backups/{id}/export_record and decodes backup_service and the base64 backup_url.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupGetCreateRestoreExportImportAndReset`.

Python/Go differences:

- Python returns the whole response JSON dict with backup_url as a string; Go returns BackupRecord with BackupURL decoded from base64 into bytes and fails if it is not valid base64.

### `python:block_storage/v2/export_record`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.Export`

- The deprecated alias sends the same GET backups/{id}/export_record that Export sends.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupGetCreateRestoreExportImportAndReset`.

Python/Go differences:

- Python emits a RemovedInSDK50Warning before delegating to export_backup; Go has no deprecated alias.
- Python returns the whole response JSON dict with backup_url as a string; Go returns BackupRecord with BackupURL decoded from base64 into bytes.

### `python:block_storage/v2/restore_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.RestoreFromBackup`

- RestoreFromBackup posts restore with volume_id and name to backups/{id}/restore and decodes backup_id, volume_id and volume_name.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupGetCreateRestoreExportImportAndReset`.

Python/Go differences:

- Python raises SDKException before HTTP when neither volume nor name is given; Go sends an empty restore object.
- Python returns the Backup updated from the restore response; Go returns Restore with BackupID, VolumeID and VolumeName. Go accepts only 202.

### `python:block_storage/v2/reset_backup_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.ResetStatus`

- ResetStatus posts os-reset_status with the status to backups/{id}/action.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupGetCreateRestoreExportImportAndReset`.

Python/Go differences:

- Go accepts only 202.
- Cinder v2 has no microversion: neither Python v2 Resources nor the Go v2 client send a volume version header unless a session default or client Microversion is configured.

### `python:block_storage/v2/reset_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups.API.ResetStatus`

- The deprecated alias sends the same os-reset_status action that ResetStatus sends.
  Tests: `blockstorage/v2/backups/python_parity_test.go:TestPythonBackupGetCreateRestoreExportImportAndReset`.

Python/Go differences:

- Python emits a RemovedInSDK50Warning before delegating to reset_backup_status; Go has no deprecated alias.
- Go accepts only 202.

### `python:block_storage/v2/create_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.WithCreateField`

- Create posts the transfer envelope with volume_id and name to os-volume-transfer and decodes id, auth_key, links and created_at.
  Tests: `blockstorage/v2/transfers/python_parity_test.go:TestPythonTransferCreateGetAcceptMatchProxyRequests`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Go requires a nonempty VolumeID and omits an empty name.
- Go accepts only 202.

### `python:block_storage/v2/get_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.Get`

- Get sends GET os-volume-transfer/{id}.
  Tests: `blockstorage/v2/transfers/python_parity_test.go:TestPythonTransferCreateGetAcceptMatchProxyRequests`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.

### `python:block_storage/v2/transfers`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.List`

- List sends GET os-volume-transfer/detail.
  Tests: `blockstorage/v2/transfers/python_parity_test.go:TestPythonTransferListAndFindUseDifferentRoutes`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/find_transfer`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.Find`

- Find(resource.ID) sends GET os-volume-transfer/{value} and Find(resource.Name) lists os-volume-transfer/detail.
  Tests: `blockstorage/v2/transfers/python_parity_test.go:TestPythonTransferListAndFindUseDifferentRoutes`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/accept_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.Accept`

- Accept posts accept.auth_key to os-volume-transfer/{id}/accept and decodes the transfer.
  Tests: `blockstorage/v2/transfers/python_parity_test.go:TestPythonTransferCreateGetAcceptMatchProxyRequests`.

Python/Go differences:

- Python sends auth_key null when it is None; Go requires a nonempty AuthKey.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Go accepts only 202.

### `python:block_storage/v2/delete_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE os-volume-transfer/{id}, ignores 404 by default, returns resource.ErrNotFound with WithMissingError and keeps 409.
  Tests: `blockstorage/v2/transfers/python_parity_test.go:TestPythonTransferDeleteIgnoresMissing`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.

### `python:block_storage/v2/get_limits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/limits.API.Get`

- Get sends GET limits and decodes the absolute and rate limits.
  Tests: `blockstorage/v2/limits/python_parity_test.go:TestPythonLimitsGetIgnoresProject`.

Python/Go differences:

- The v2 Limits resource has no project URI or query mapping, so Python get_limits(project) also sends plain GET limits; Go takes no project argument.
- Python returns Limits resources; Go returns typed Absolute and Rate values.

### `python:block_storage/v2/get_quota_set`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.GetUsage`

- Get sends GET os-quota-sets/{project} without a query and GetUsage sends usage=true.
  Tests: `blockstorage/v2/quotasets/python_parity_test.go:TestPythonQuotaSetGetAndRevertCannotSendQuery`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_quota_set_defaults`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.Get`

- Get(ctx, "defaults") sends GET os-quota-sets/defaults like the pinned base_path override, while GetDefaults sends the project-scoped route.
  Tests: `blockstorage/v2/quotasets/python_parity_test.go:TestPythonQuotaSetDefaultsUsesUnscopedRoute`.

Python/Go differences:

- The pinned v2 proxy ignores the project for the URL and fetches os-quota-sets/defaults; Go reproduces it with Get(ctx, "defaults"), while quotasets.API.GetDefaults sends os-quota-sets/{project}/defaults.
- Python returns a QuotaSet with normalized attributes; Go returns QuotaSet with per-volume-type keys in Extra.

### `python:block_storage/v2/update_quota_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.WithUpdateOptions`

- Update puts the quota_set envelope with the given limits and extra per-type keys to os-quota-sets/{project} and decodes the response.
  Tests: `blockstorage/v2/quotasets/python_parity_test.go:TestPythonQuotaSetUpdateSendsDirtyAttributes`.

Python/Go differences:

- Python sends no request when no attribute is dirty; Go Update always sends PUT, even with an empty envelope.
- Python accepts a Project or QuotaSet and the deprecated project_id/query arguments with warnings; Go takes the project ID string.
- Go accepts only 200.

### `python:block_storage/v2/revert_quota_set`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets.API.Delete`

- Delete sends DELETE os-quota-sets/{project} without a query and treats 204 as an error.
  Tests: `blockstorage/v2/quotasets/python_parity_test.go:TestPythonQuotaSetGetAndRevertCannotSendQuery`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_quota_class_set`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/update_quota_class_set`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/backend_pools`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/schedulerstats.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/schedulerstats.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/schedulerstats.WithListOptions`

- List with WithListQuery("detail", "True") sends scheduler-stats/get_pools?detail=True, the typed Detail option sends detail=true, and pool names and typed capabilities decode.
  Tests: `blockstorage/v2/schedulerstats/python_parity_test.go:TestPythonBackendPoolsSendsDetailQuery`.

Python/Go differences:

- Python keeps the whole capabilities dict; Go Capabilities decodes a fixed field set, turns infinite capacities into +Inf and other nonnumeric capacities into 0.
- Python filters Body attributes outside the query mapping locally; Go leaves local filtering to the caller.

### `python:block_storage/v2/get_capabilities`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/services.WithListOptions`

- List sends GET os-services with binary and host queries and decodes binary, host, zone, state, frozen, replication_status and updated_at.
  Tests: `blockstorage/v2/services/python_parity_test.go:TestPythonServicesListFiltersByBinaryAndHost`.

Python/Go differences:

- Python exposes zone as availability_zone, binary also as name and frozen as is_frozen; Go keeps the wire names in Service fields.
- Python filters Body attributes outside the binary/host query mapping locally; Go leaves local filtering to the caller.

### `python:block_storage/v2/find_service`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/enable_service`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/disable_service`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/thaw_service`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/freeze_service`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/failover_service`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/extensions`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_type`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/find_type`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/types`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/create_type`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/update_type`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/delete_type`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/update_type_extra_specs`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/delete_type_extra_specs`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_type_access`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/add_type_access`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/remove_type_access`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_type_encryption`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/create_type_encryption`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/update_type_encryption`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/delete_type_encryption`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/create_qos_spec`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_qos_spec`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/find_qos_spec`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/qos_specs`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/update_qos_spec`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/delete_qos_spec`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/associate_qos_spec`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/disassociate_qos_spec`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/disassociate_all_qos_spec`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/qos_spec_associations`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_consistency_group`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/find_consistency_group`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/consistency_groups`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/create_consistency_group`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/create_consistency_group_from_source`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/update_consistency_group`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/delete_consistency_group`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

## References

- [block_storage v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1026-L1226)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1902-L2029)
- [`get_limits`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1230-L1248)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1516-L1622)
- [`backend_pools`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1011-L1022)
- [`services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1671-L1681)
- [Python Parity](../v3/python-parity.md)
- [Consistency Groups Python](../v3/consistency-groups-python.md)
