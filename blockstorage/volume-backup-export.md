# Volume Backup Export

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/export_volume_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ExportVolumeBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ExportVolumeBackup`

- Required BackupID is one safe nonempty literal UTF-8 member segment. Export requests bodyless queryless GET backups/{ID}/export_record without member/list/Volume/Snapshot/Identity resolution, generated identity, wait or SDK workflow replay. Unicode is escaped once; result retains the requested ID string rather than a response-derived identity.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupOpaqueSub400ResponseUsesOneBodylessFixedGETAndSelectedMicroversion`, `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupPreflightCapturesSourceBeforeSafeIDAndNeverMakesLookupRequests`, `connection_volume_backup_export_test.go:TestConnectionBackupExportPreflightAndSafeIDRejectBeforeCinder`, `connection_volume_backup_export_test.go:TestConnectionBackupExportOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`.
- The original finite HTTP 100..399 policy preserves final opaque body bytes, headers and status without JSON, UTF-8, envelope, Backup descriptor or native base64/typed-record parsing. Selected service microversion, including omission, is unchanged; no automatic 3.64 override, default, discovery or cap is added.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupOpaqueSub400ResponseUsesOneBodylessFixedGETAndSelectedMicroversion`, `connection_volume_backup_export_test.go:TestConnectionBackupExportOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`.
- Exported Body, multivalue Header and StatusCode form an independently owned physical response snapshot. Different calls and physical fixtures do not alias, and error response evidence is separately owned. Full requests.Response object identity is an intentional Go mapping.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupOwnsOpaqueBodyAndHeadersAcrossIndependentCalls`, `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupAcceptedIOCancellationAndSourceFailuresKeepCurrentOwnedOpaqueProof`, `connection_volume_backup_export_test.go:TestConnectionBackupExportOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`, `connection_volume_backup_export_test.go:TestConnectionBackupExportAcceptedOpaqueIOOwnsCurrentProofAndAllCauses`.
- Direct calls capture and validate the selected Cinder source before literal ID validation. Connection validates context/provider/ID before selecting cached Cinder v3 and passes service.RawClient; no CurrentLocation or Backup model/readiness is consumed. Invalid context/ID/source/getter is local, with outer ExportVolumeBackup/volume backup operation and caller causes preserved.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupPreflightCapturesSourceBeforeSafeIDAndNeverMakesLookupRequests`, `connection_volume_backup_export_test.go:TestConnectionBackupExportPreflightAndSafeIDRejectBeforeCinder`, `connection_volume_backup_export_test.go:TestConnectionBackupExportOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`, `connection_volume_backup_export_test.go:TestConnectionBackupExportGetterFailuresJoinCustomAndContextCauses`.
- Admitted response proof is published before accepted Read/Close/source/context errors are returned. Current partial bytes, status and headers remain in Exported with joined caller causes and independently owned ResponseError. Exported presence alone does not establish success; accepted faults do not trigger retry, SDK replay or rollback.
  Tests: `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupAcceptedIOCancellationAndSourceFailuresKeepCurrentOwnedOpaqueProof`, `connection_volume_backup_export_test.go:TestConnectionBackupExportAcceptedOpaqueIOOwnsCurrentProofAndAllCauses`.
- Original native 400/403/404 and other rejected >=400 responses are terminal with native evidence and Exported nil; no lookup fallback or normal absence is introduced. Native expanded OkCodes cannot fabricate admitted proof outside the original policy. Rejected response body IO retains native discarded Read/Close limitations, independent of member/poll response-body observers.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupNativeRejectionsAreTerminalAndDoNotInventAdmittedProof`, `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupNativeExpandedStatusCannotCreateAdmittedProof`.
- Live provider authentication, native 401 reauthentication and bounded caller retry retain the fixed request and captured ordinary source headers while allowing existing native header hooks. Export changes no selected microversion/client state; cached Connection calls consume no CurrentLocation and use later live tokens and later call header snapshots.
  Tests: `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupNativeReauthAndBoundedRetryUseLiveTokensAndOwnedHeaders`, `connection_volume_backup_export_test.go:TestConnectionBackupExportOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`.
- Fixed method/URL, native option bodylessness/body retention/decoder policy and five sticky service facts remain SDK-owned through retries and redirects. Export also rejects physical nonnil Body other than http.NoBody, nonzero ContentLength and nonempty TransferEncoding before underlying transport. That physical body rejection remains terminal after native retry restores the request, preserving original rejection, callback and cancellation causes without fabricated admitted proof. Changed-path/foreign-origin redirects stop before a new physical request or token transmission; no SDK workflow replay occurs.
  Tests: `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupRejectsNativeRetryBodyDecoderRetentionOrSourceMutationBeforeResend`, `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupRedirectCannotChangeFixedTargetOrSendTokenToAnotherOrigin`, `blockstorage/backup_export_redirect_body_test.go:TestExportVolumeBackupSameTargetRedirectCannotAddPhysicalBodyFraming`, `blockstorage/backup_export_redirect_body_test.go:TestExportVolumeBackupRedirectBodyRejectionSurvivesNativeRetryRestorationAndJoinedCauses`.

Python/Go differences:

- Only actual direct v3 BlockStorageCloudMixin.export_volume_backup is mapped. Independently parsed Proxy.export_backup/Backup.export_record, deprecated aliases, native typed Export/base64 BackupRecord, v2 dict-return path and remaining Backup actions retain independent reviews. Full SDK remains incomplete.
- Python dynamic resource/dict/Munch/None and permissive urljoin falsey/container/leading-slash behavior map to a safe single literal UTF8 Go BackupID string. No NameOrID resolver, generatedID or Resource._get_id shortcut is introduced.
- Source Proxy._get_resource creates or updates a connected mutable Backup with incidental constructor location/descriptor/to_dict side effects. ID-only Go export deliberately skips those effects and consumes no configured CurrentLocation; these constructor failures/caches/identity are not replicated.
- v3 returns the same requests.Response without response.json. Go snapshots actual raw Body/Header/StatusCode rather than mutable Response request/history/cookies/encoding/elapsed/methods and deprecated warnings. Numeric spelling and arbitrarybinarybytes remain physical data.
- The source has no options or wait policy. Go adds parent context interruption/customcause and ownedpartialproof, not empty With/Prepare builders, wait/location options, readinesspoll or SDKtimeout. Selected microversion is preserved without automatic discovery/cap or forceaction3.64.
- Fixed same-origin method/route/request-policy and sticky source guards constrain redirects and native callbacks. A closed export-only physical guard also rejects Body, ContentLength and TransferEncoding mutations after redirect hooks; it does not alter old shared request code or apply rejected response-body IO observers. Existing native ordinary header hooks remain; this is not force-style per-header microversion enforcement.
- Accepted Read/Close/context/source failures retain actual admitted proof and simultaneous causes. Originalnative or originalpolicy>=400 rejection has no Exported; native rejectedbodyRead/Close discardedfault scope remains, not a universal IOobserver repair.
- FiniteHTTP100..399 mirrors source<400 in realHTTPdomain. Local source/probe evidence uses frozen literals and explicitly labeled AST/session/response stand-ins; authenticated Python/OpenStack execution is not claimed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Adopt(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
    return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func Inspect(result *blockstorage.ExportVolumeBackupResult, err error) {
    if result != nil {
        fmt.Println("requested backup ID:", result.BackupID)
        if result.Exported != nil {
            response := result.Exported
            fmt.Println("export HTTP:", response.StatusCode, "bytes:", len(response.Body))
            fmt.Println("content type:", response.Header.Get("Content-Type"))
        }
    }
    if err == nil { return }
    fmt.Println("export error:", err)
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("parent context interrupted export")
    }
    if errors.Is(err, resource.ErrInvalidOption) {
        fmt.Println("invalid local input or changed request source")
    }
}

func ConfiguredExport(parent context.Context, cloudName, backupID string) (*blockstorage.ExportVolumeBackupResult, error) {
    ctx, cancel := context.WithTimeout(parent, 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return nil, err }
    result, err := conn.ExportVolumeBackup(ctx,
        blockstorage.ExportVolumeBackupRequest{BackupID: backupID})
    Inspect(result, err)
    return result, err
}

func DirectExport(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*blockstorage.ExportVolumeBackupResult, error) {
    result, err := blockstorage.ExportVolumeBackup(ctx, cinder,
        blockstorage.ExportVolumeBackupRequest{BackupID: backupID})
    Inspect(result, err)
    return result, err
}

// Optional application policy: call after a successful export only when
// the application expects JSON. An opaque export need not contain JSON.
func ApplicationJSON(result *blockstorage.ExportVolumeBackupResult) (json.RawMessage, error) {
    if result == nil || result.Exported == nil {
        return nil, fmt.Errorf("no admitted export response")
    }
    var value json.RawMessage
    if err := json.Unmarshal(result.Exported.Body, &value); err != nil {
        return nil, err
    }
    return value, nil
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack", block_storage_api_version="3")
response = conn.export_volume_backup(backup_id)
print(response.status_code, response.headers, response.content)
```

## References

- [ Block Storage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L648)
- [deprecated v3 `Proxy.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1824)
- [`Backup.export`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L184)
- [`Proxy.export_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1812)
- [`Backup.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L173)
- [JSON result](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L281)
- [typed `BackupRecord`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/results.go#L201)
- [Volume Backups](volume-backups.md)
- [Volume Backup Mutations](volume-backup-mutations.md)
- [..](../README.md)
