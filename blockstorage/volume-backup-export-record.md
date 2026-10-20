# Volume Backup Export Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/export_record`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ExportRecord`

- API.ExportRecord requires context and one safe nonempty literal UTF-8 backup ID, escaped once; names remain literal IDs. The captured raw engine validates source before ID and performs no member/list/Identity resolution or generated identity. Python Resource/dict/Munch/None construction and local location/descriptor side effects map to the explicit Go string-ID boundary.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupPreflightCapturesSourceBeforeSafeIDAndNeverMakesLookupRequests`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesPreflightKeepOuterMethodContext`.
- The raw operation requests bodyless queryless GET backups/{ID}/export_record through one captured source without wait, model normalization, cleanup, rollback or SDK workflow replay. Fixed method/URL/source and physical Body/ContentLength/TransferEncoding policy remain guarded through redirects and native callbacks.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupOpaqueSub400ResponseUsesOneBodylessFixedGETAndSelectedMicroversion`, `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupRedirectCannotChangeFixedTargetOrSendTokenToAnotherOrigin`, `blockstorage/backup_export_redirect_body_test.go:TestExportVolumeBackupSameTargetRedirectCannotAddPhysicalBodyFraming`, `blockstorage/backup_export_redirect_body_test.go:TestExportVolumeBackupRedirectBodyRejectionSurvivesNativeRetryRestorationAndJoinedCauses`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesSeparateOpaqueJSONAndLegacyTypedProjection`.
- Keep selected microversion including omission, captured ordinary source headers and existing live token/native reauthentication/bounded caller retry. No forced 3.64, discovery, default or cap is added. The raw engine consumes no CurrentLocation and never recaptures resource scope.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupOpaqueSub400ResponseUsesOneBodylessFixedGETAndSelectedMicroversion`, `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupNativeReauthAndBoundedRetryUseLiveTokensAndOwnedHeaders`, `connection_volume_backup_export_test.go:TestConnectionBackupExportOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesSeparateOpaqueJSONAndLegacyTypedProjection`.
- Original finite HTTP 100..399 responses return an independently owned opaque Body/Header/StatusCode snapshot, including binary, invalid UTF-8, malformed/nonobject JSON and empty 204, without JSON or native base64/BackupRecord projection. Result BackupID is the requested string.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupOpaqueSub400ResponseUsesOneBodylessFixedGETAndSelectedMicroversion`, `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupOwnsOpaqueBodyAndHeadersAcrossIndependentCalls`, `connection_volume_backup_export_test.go:TestConnectionBackupExportOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesSeparateOpaqueJSONAndLegacyTypedProjection`.
- Accepted Read/Close/source/context errors retain current Exported and independently owned ResponseError evidence, simultaneous caller causes and outer operation wrapping; Exported alone is not success. Rejected >=400 including 404 or failed redirect has no Exported and no lookup/absence fallback. Rejected response-body IO retains native limitations; no universal observer claim.
  Tests: `blockstorage/backup_export_contract_test.go:TestExportVolumeBackupNativeRejectionsAreTerminalAndDoNotInventAdmittedProof`, `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupAcceptedIOCancellationAndSourceFailuresKeepCurrentOwnedOpaqueProof`, `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupNativeExpandedStatusCannotCreateAdmittedProof`, `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupRedirectCannotChangeFixedTargetOrSendTokenToAnotherOrigin`, `connection_volume_backup_export_test.go:TestConnectionBackupExportAcceptedOpaqueIOOwnsCurrentProofAndAllCauses`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesPreflightKeepOuterMethodContext`.
- Retry body/decoder/retention or five captured service fact changes, and physical redirected request body/framing mutation, stop underlying resend with sticky policy errors. Restoration or swallowed callback errors cannot erase physical body rejection. Original rejection, callback and context.Cause remain reachable without fabricated admitted proof.
  Tests: `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupRejectsNativeRetryBodyDecoderRetentionOrSourceMutationBeforeResend`, `blockstorage/backup_export_transport_test.go:TestExportVolumeBackupRedirectCannotChangeFixedTargetOrSendTokenToAnotherOrigin`, `blockstorage/backup_export_redirect_body_test.go:TestExportVolumeBackupSameTargetRedirectCannotAddPhysicalBodyFraming`, `blockstorage/backup_export_redirect_body_test.go:TestExportVolumeBackupRedirectBodyRejectionSurvivesNativeRetryRestorationAndJoinedCauses`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesPreflightKeepOuterMethodContext`.
- The manual API.ExportRecord facade delegates to the existing raw engine, owns ExportRecord/backups outer error context, retains inner causes and physical result, and handles nil API as a local missing-client failure. The actual source target is deprecated v3 Proxy.export_record→Backup.export; Python warnings/filter-as-error are intentionally omitted and existing native API.Export is an independent typed operation.
  Tests: `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesPreflightKeepOuterMethodContext`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesSeparateOpaqueJSONAndLegacyTypedProjection`.

Python/Go differences:

- Only actual v3 Proxy.export_record is this target. Its bodyless raw path is shared with separately reviewed cloud export_volume_backup. Parsed Proxy.export_backup/Resource Backup.export_record, deprecated Backup.export, native typed Export and v2 retain independent declaration reviews; full SDK remains incomplete.
- Python permits connected mutable Resource, dict/Munch, None and permissive urljoin falsey/container/leading-slash identity. Go requires one safe literal UTF-8 string ID and intentionally omits Resource construction/update/current_location/descriptor/cache/object-identity side effects.
- Both deprecated Proxy and Backup source warnings, including warning filters that can raise before construction or HTTP, are omitted from Go. API.ExportRecord/backups outer OperationError wraps the shared ExportVolumeBackup/volume backup engine while retaining causes.
- The same mutable requests.Response maps to owned actual Body/Header/StatusCode rather than request/history/cookies/encoding/elapsed/response convenience methods. No UTF-8/JSON/base64/schema requirement is applied to raw bytes.
- Go adds parent context interruption, joined custom causes and separately owned accepted partial/error proof. Original/native >=400 has no Exported, and native rejected response Read/Close limitations remain; no generic response-body observer promotion.
- Safe fixed route/method/option policy and export-only physical body/framing guard constrain mutable native hooks/redirects. Native ordinary header hooks, live token and native retry/reauth remain; there is no force-style 3.64 override or universal header enforcement.
- Finite real HTTP100..399 maps source <400. Source literals and local tests are distinct evidence; no authenticated Python/OpenStack execution, exhaustive HTTP/runtime-object domain or generic transport/Resource parity claim.

### `python:block_storage/v3/export_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ExportBackup`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ExportVolumeBackupRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ExportVolumeBackupRecord`

- ExportVolumeBackupRecord package and Connection plus API.ExportBackup accept one safe nonempty literal UTF-8 ID with no resolver, generated identity, options, wait or CurrentLocation conversion. Direct/service engine captures selected source before ID; Connection performs local context/provider/ID preflight before cached Cinder v3 selection.
  Tests: `blockstorage/backup_export_record_contract_test.go:TestExportVolumeBackupRecordPreflightUsesCapturedSourceAndSafeLiteralID`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordPreflightAndSafeIDRejectBeforeCinder`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordGetterFailuresJoinCustomAndContextCauses`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesPreflightKeepOuterMethodContext`.
- One captured reader owns bodyless queryless GET backups/{ID}/export_record, pure parse and final source check; source is not recaptured after response. Original HTTP100..399 and selected microversion including omission remain, without forced3.64/discovery/cap. Live token/native authentication/retry and fixed request policies are shared with raw exchange.
  Tests: `blockstorage/backup_export_record_contract_test.go:TestExportVolumeBackupRecordAcceptsArbitraryJSONAndPreservesLiteralValue`, `blockstorage/backup_export_record_transport_test.go:TestExportVolumeBackupRecordNativeReauthAndBoundedRetryUseLiveTokensAndOwnedHeaders`, `blockstorage/backup_export_record_transport_test.go:TestExportVolumeBackupRecordRejectsNativeRetryBodyDecoderRetentionOrSourceMutationBeforeResend`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesSeparateOpaqueJSONAndLegacyTypedProjection`.
- Value is one whole arbitrary JSON value without backup-record extraction, runtime dict/object assertion, base64 projection or Backup descriptor conversion. Object/array/string/number/bool/null and falsey/empty values are valid; successful null is nonnil bytes null, while no published Value is nil.
  Tests: `blockstorage/backup_export_record_contract_test.go:TestExportVolumeBackupRecordAcceptsArbitraryJSONAndPreservesLiteralValue`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesSeparateOpaqueJSONAndLegacyTypedProjection`.
- The chosen Go representation is independently owned literal UTF-8 json.RawMessage: document leading/trailing whitespace excluded; interior whitespace, number spelling/precision, escapes and duplicate member literals retained. No float64/UseNumber tree or serialization normalization is introduced. This is explicitly different from semantic mutable Python dict/list/scalar decoding.
  Tests: `blockstorage/backup_export_record_contract_test.go:TestExportVolumeBackupRecordAcceptsArbitraryJSONAndPreservesLiteralValue`, `blockstorage/backup_export_record_contract_test.go:TestExportVolumeBackupRecordValuePhysicalProofAndSeparateCallsOwnDifferentStorage`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesSeparateOpaqueJSONAndLegacyTypedProjection`.
- A complete ordinary UTF-8 JSON document and EOF are required. Empty/malformed/trailing-second-value/trailing-garbage, unsupported UTF16/UTF32/BOM/invalidUTF8 and nonfinite NaN/Infinity tokens fail parsing without another HTTP request or native JSONResponse retry. Original rejected HTTP errors precede JSON parsing and never invent Exported/Value.
  Tests: `blockstorage/backup_export_record_contract_test.go:TestExportVolumeBackupRecordRejectsInvalidDocumentWithoutParserRetryAndKeepsActualProof`, `blockstorage/backup_export_record_contract_test.go:TestExportVolumeBackupRecordHTTPRejectionPrecedesJSONParsingAndHasOnlyNativeProof`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordParserFailureRetainsOpaqueProofWithParsedOperationAndNoNativeRetry`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceParsedFailuresKeepCurrentProofAndNeverRetryDecoder`.
- Actual opaque Exported is copied before accepted IO/source/context or JSON failures. Value is published only after full parse and last guard on the same captured source succeeds. Partial/current physical bytes/header/status and joined custom causes remain separately owned; Value, Exported and ResponseError do not alias. Exported presence alone does not establish success.
  Tests: `blockstorage/backup_export_record_contract_test.go:TestExportVolumeBackupRecordRejectsInvalidDocumentWithoutParserRetryAndKeepsActualProof`, `blockstorage/backup_export_record_contract_test.go:TestExportVolumeBackupRecordValuePhysicalProofAndSeparateCallsOwnDifferentStorage`, `blockstorage/backup_export_record_transport_test.go:TestExportVolumeBackupRecordAcceptedIOCancellationAndSourceFailuresKeepCurrentOwnedOpaqueProof`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordAcceptedOpaqueIOOwnsCurrentProofAndAllCauses`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordParserFailureRetainsOpaqueProofWithParsedOperationAndNoNativeRetry`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceParsedFailuresKeepCurrentProofAndNeverRetryDecoder`.
- Bodyless/decoder/retention/method/URL and sticky five-fact source policy apply through native retry/reauth and redirects, including physical Body/ContentLength/TransferEncoding rejection before transport and terminal rejection after restoration. Expanded native OkCodes cannot admit original-policy >=400; safe changed-path/foreign-origin redirects cannot resend credentials. No SDK replay/rollback occurs.
  Tests: `blockstorage/backup_export_record_transport_test.go:TestExportVolumeBackupRecordRejectsNativeRetryBodyDecoderRetentionOrSourceMutationBeforeResend`, `blockstorage/backup_export_record_transport_test.go:TestExportVolumeBackupRecordNativeExpandedStatusCannotCreateAdmittedProof`, `blockstorage/backup_export_record_transport_test.go:TestExportVolumeBackupRecordRejectsPhysicalRedirectMutationAndKeepsStickyCause`, `blockstorage/backup_export_redirect_body_test.go:TestExportVolumeBackupSameTargetRedirectCannotAddPhysicalBodyFraming`, `blockstorage/backup_export_redirect_body_test.go:TestExportVolumeBackupRedirectBodyRejectionSurvivesNativeRetryRestorationAndJoinedCauses`.
- Connection exposes ExportVolumeBackupRecord and cached Cinder v3 only; direct package and service ExportBackup retain their exact public/outer operation context and underlying causes, including nil API local failure. The actual source graph is v3 Proxy.export_backup→Backup.export_record→response.json; warning/Resource/session/backend/encoding identities are explicit Go mappings, independently of raw and native/v2/Resource review status.
  Tests: `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordPreflightAndSafeIDRejectBeforeCinder`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordGetterFailuresJoinCustomAndContextCauses`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordAcceptedOpaqueIOOwnsCurrentProofAndAllCauses`, `connection_volume_backup_export_record_test.go:TestConnectionBackupExportRecordParserFailureRetainsOpaqueProofWithParsedOperationAndNoNativeRetry`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesPreflightKeepOuterMethodContext`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceFacadesSeparateOpaqueJSONAndLegacyTypedProjection`, `blockstorage/backup_proxy_export_service_test.go:TestBackupProxyExportServiceParsedFailuresKeepCurrentProofAndNeverRetryDecoder`.

Python/Go differences:

- Only actual v3 Proxy.export_backup is this target. ExportVolumeBackupRecord is a Go convenience spelling for this Proxy operation, not a new Python cloud declaration. Raw Proxy.export_record, Backup.export_record method, native typed/base64 Export, v2 and generic Resource remain independent reviews; full SDK is incomplete.
- Go uses a safe literal UTF-8 BackupID string and skips Python _get_resource construction/update of connected Resource/dict/Munch/None, mutable descriptor/location/cache/object identity and permissive urljoin coercion. Connection consumes no CurrentLocation.
- Python response.json returns a semantic backend/interpreter object. Go preserves a validated one-document UTF-8 literal RawMessage of any shape, with number/escape/duplicate text and interior spacing intact; document exterior whitespace is excluded. Duplicate last-value semantics, float rounding/underflow, unpaired string surrogate interpretation and interpreter/backend number/depth limits are not replicated. Value must be explicitly decoded by callers who need a semantic Go structure.
- The SDK source pin does not lock Requests/simplejson/interpreter versions or selected decoder. Requests v2.32.5 references show optional simplejson/stdlib, encoding detection including UTF16/32/BOM and text replacement fallback; those references are not SDK exact dependency pins. Go accepts only ordinary UTF8 JSON, rejects unsupported encoding/BOM/nonfinite tokens and applies Go parser structural limits.
- Source annotations/cast do not enforce a dict, BackupRecord envelope or base64; Go likewise admits arbitrary JSON shapes. Source .json occurs after session.get, so parsing is local after the opaque exchange and never causes another HTTP attempt. Existing native JSONResponse/BackupRecord projection remains untouched.
- Go adds raw admitted Exported and joined partial IO/context/source/parser proof, with nil Value on any failure and nonnil null on successful JSON null. Parent context, independently owned result/error bytes/header and operation wrapping replace Python exception/session/Response identities. Native rejected body Read/Close discarded errors remain a bounded limitation.
- Safe fixed route/method/native option and export-only physical body/framing policy constrain mutable native callbacks/redirects. Ordinary native header hooks and live authentication/retry remain; source/body rejection is sticky, selected microversion unchanged and no 3.64 forced action/default/cap/discovery is added.
- Parsed source path contains neither the raw Proxy nor Backup deprecated warning emission; Go does not recreate the Python warning machinery. Local source/tests/docs gates and backend references do not establish authenticated Python/OpenStack runtime or automatic native/v2/Resource parity.

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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func Inspect(response *blockstorage.VolumeBackupExportResponse, value json.RawMessage, err error) {
    if response != nil {
        fmt.Println("export HTTP:", response.StatusCode, "bytes:", len(response.Body))
        fmt.Println("content type:", response.Header.Get("Content-Type"))
    }
    if err != nil {
        fmt.Println("export error:", err)
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("parent context interrupted export")
        }
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("accepted error proof:", physical.StatusCode, len(physical.Body))
        }
        return
    }
    if value != nil {
        // Literal null is successful nonnil bytes "null"; nil means no Value.
        fmt.Printf("JSON record: %s\n", value)
    }
}

func ConfiguredRecord(parent context.Context, cloudName, backupID string) (*blockstorage.ExportVolumeBackupRecordResult, error) {
    ctx, cancel := context.WithTimeout(parent, 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return nil, err }
    result, err := conn.ExportVolumeBackupRecord(ctx,
        blockstorage.ExportVolumeBackupRecordRequest{BackupID: backupID})
    if result != nil { Inspect(result.Exported, result.Value, err) }
    return result, err
}

func ServiceRecord(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.ExportBackupResult, error) {
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    result, err := cinder.Backups.ExportBackup(ctx, backupID)
    if result != nil { Inspect(result.Exported, result.Value, err) }
    return result, err
}

func ServiceRaw(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.ExportRecordResult, error) {
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    result, err := cinder.Backups.ExportRecord(ctx, backupID)
    if result != nil { Inspect(result.Exported, nil, err) }
    return result, err
}

func DirectRecord(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*blockstorage.ExportVolumeBackupRecordResult, error) {
    result, err := blockstorage.ExportVolumeBackupRecord(ctx, cinder,
        blockstorage.ExportVolumeBackupRecordRequest{BackupID: backupID})
    if result != nil { Inspect(result.Exported, result.Value, err) }
    return result, err
}

func DirectServiceRecord(ctx context.Context, cinder *gophercloud.ServiceClient, backupID string) (*backups.ExportBackupResult, error) {
    return backups.New(cinder).ExportBackup(ctx, backupID)
}

// Native typed Export is an alternative for the native BackupRecord schema.
func NativeTyped(ctx context.Context, conn *sdk.Connection, backupID string) (*backups.BackupRecord, error) {
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    return cinder.Backups.Export(ctx, backupID)
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack", block_storage_api_version="3")
value = conn.block_storage.export_backup(backup_id)
print(value)

response = conn.block_storage.export_record(backup_id)
print(response.status_code, response.headers, response.content)
```

## References

- [`Proxy.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1824)
- [`Proxy.export_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1812)
- [`Backup.export_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L173)
- [`Backup.export`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L184)
- [Response.json](https://github.com/psf/requests/blob/v2.32.5/src/requests/models.py#L947)
- [compat](https://github.com/psf/requests/blob/v2.32.5/src/requests/compat.py#L57-L64)
- [guess_json_utf](https://github.com/psf/requests/blob/v2.32.5/src/requests/utils.py#L947)
- [`backups.Export`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L281)
- [`BackupRecord`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/results.go#L201)
- [Cloud raw export](volume-backup-export.md)
- [Volume Backups](volume-backups.md)
- [Volume Backup Mutations](volume-backup-mutations.md)
- [..](../README.md)
