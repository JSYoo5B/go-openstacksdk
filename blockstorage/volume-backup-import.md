# Volume Backup Import

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/import_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ImportBackup`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ImportVolumeBackup`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ImportVolumeBackup`

- Package, Connection and service import take two explicit literal UTF-8 strings, including empty/control/Unicode values, without defaults, enum, URI/base64 validation, resolver, builder or wait. One queryless POST backups/import_record contains exactly backup-record.backup_service and the caller backup_url. This explicitly repairs the pinned Source overwritten export_record route and argument. Caller composition with a parsed export locator adds no automatic bytes conversion.
  Tests: `blockstorage/backup_import_contract_test.go:TestImportVolumeBackupRepairsRouteAndForwardsBothLiteralStrings`, `blockstorage/backup_import_contract_test.go:TestImportVolumeBackupComposesWithParsedExportWithoutBase64Transformation`.
- Direct source capture and context/provider/endpoint/resource/header validation precede input admission. Invalid UTF-8 fails locally. Connection validates context/provider/input before selecting only cached Cinder v3; getter errors retain outer operation and every custom cancellation cause without fabricated HTTP proof.
  Tests: `blockstorage/backup_import_contract_test.go:TestImportVolumeBackupPreflightRejectsOnlyInvalidUTF8AndInvalidCapturedSource`, `connectiontest/connection_volume_backup_import_test.go:TestConnectionImportVolumeBackupPreflightRejectsBeforeOnlyCinderSelection`, `connectiontest/connection_volume_backup_import_test.go:TestConnectionImportVolumeBackupGetterFailuresJoinCallerCausesWithoutProof`.
- Selected nonempty client microversion wins literally with no discovery/cap or original-client mutation. Otherwise finite same-origin captured-endpoint bodyless discovery preserves encoded reverse-proxy prefixes and selects the first usable v3 row from supported flat/version/versions/versions.values shapes. Maximum is parsed before optional minimum, capped against3.64 using tuple order/length; missing maximum ignores unused minimum, and minimum above cap omits version. Signed arbitrary-precision Unicode decimal/underscore strings and finite-major latest header failure follow the owned bounded parser policy.
  Tests: `blockstorage/backup_import_discovery_test.go:TestImportVolumeBackupSelectedMicroversionWinsWithoutDiscoveryOrCap`, `blockstorage/backup_import_discovery_test.go:TestImportVolumeBackupDiscoverySelectsBoundedMaximumAndOptionalMinimum`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentStringAndHeaderCases`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentMaximumMinimumCases`.
- Only clean concrete native404/405 allows finite candidate fallback; accepted empty/no usable v3 may advance candidates, while a found v3 row without maximum ends version selection. Accepted discovery HTTP200/300 proof is independent of import Applied. Malformed accepted data, joined rejected IO/callback faults, expanded OkCodes, five-fact source drift and custom cancellation are terminal and cannot authorize POST or fabricate admitted proof.
  Tests: `blockstorage/backup_import_discovery_test.go:TestImportVolumeBackupDiscoveryIsFiniteAndCleanUnavailableFallsBack`, `blockstorage/backup_import_discovery_test.go:TestImportVolumeBackupInvalidAcceptedDiscoveryKeepsOnlyCurrentDiscoveryProof`, `blockstorage/backup_import_discovery_test.go:TestImportVolumeBackupDiscoveryFaultsSourceDriftAndCancellationAreTerminal`.
- A fresh disconnected nullable24 Backup has literal null location and never consumes CurrentLocation, configured/token project lookup, response zone/location or request-ID seed. Response ID remains untyped, including missing/null/false/zero/empty/container values. Known-only overlay shares ordinary Boolean/list/dict/integer descriptors; actual raw unknown/member fields remain independent evidence. Connection reuses cached Cinder and live token with captured ordinary headers, while each discovered microversion belongs to that call rather than the cached client.
  Tests: `blockstorage/backup_import_contract_test.go:TestImportVolumeBackupBuildsDisconnectedNullableBackupFromActualKnownFields`, `blockstorage/backup_import_contract_test.go:TestImportVolumeBackupKeepsArbitraryResponseIDWithoutInventingSeed`, `internal/cloudsnapshot/backup_disconnected_test.go:TestBackupDisconnectedModelKeepsNullLocationAndNeverSeedsIdentity`, `internal/cloudsnapshot/backup_disconnected_test.go:TestBackupDisconnectedModelSharesNullableDescriptorConversion`, `connectiontest/connection_volume_backup_import_test.go:TestConnectionImportVolumeBackupUsesCachedCinderAndIgnoresInvalidLocation`, `connectiontest/connection_volume_backup_import_test.go:TestConnectionImportVolumeBackupDiscoveryPolicyIsPerCallAndDoesNotMutateCache`.
- Present backup envelope wins; absent envelope selects a flat object. Ordinary empty/malformed UTF-8 JSON retains the all-null view without an invented raw object; genuine empty object remains actual evidence. Valid wrong shapes, invalid UTF-8 and descriptor conversion errors keep current accepted proof and no Value or decoder retry. A consumed physical response ID survives later descriptor failure; failures before overlay retain null.
  Tests: `blockstorage/backup_import_contract_test.go:TestImportVolumeBackupUsesEnvelopePresenceAndOnlyMalformedJSONLeavesDefaults`.
- Publish logical Value only after translation, descriptor conversion and final captured-source guard. BackupID, Value, selected raw Backup, Applied bytes/headers and separate calls own independent storage. Native original-policy >=400 rejection has no Applied or logical success. Accepted Read/Close/source/context errors keep actual stage proof and independent ResponseError with all causes, close once, and do not replay; earlier admitted discovery is never borrowed as import Applied.
  Tests: `blockstorage/backup_import_contract_test.go:TestImportVolumeBackupLogicalPhysicalAndSeparateCallsOwnIndependentStorage`, `blockstorage/backup_import_contract_test.go:TestImportVolumeBackupHTTPRejectionNeverBecomesAppliedOrLogicalSuccess`, `blockstorage/backup_import_transport_test.go:TestImportVolumeBackupAcceptedFaultsKeepAppliedAndEveryCauseWithoutValue`, `connectiontest/connection_volume_backup_import_test.go:TestConnectionImportVolumeBackupExecutionErrorsKeepOnlyActualStageProof`.
- Live native reauthentication and bounded caller retry preserve captured ordinary headers, source and owned serialized literal body. Mutable body/decoder/retention options, source changes or context interruption stop before resend with original and callback causes. Nonempty operation microversion owns consistent modern/legacy headers; explicit empty policy rejects injected versions. Retry map aliases may agree, while physical headers require one value per name.
  Tests: `blockstorage/backup_import_transport_test.go:TestImportVolumeBackupLiveReauthAndBoundedRetryKeepLiteralOwnedBody`, `blockstorage/backup_import_transport_test.go:TestImportVolumeBackupRetryMutationCannotSendChangedTargetBodyOrVersion`.
- Every physical POST, including same-target307 redirects, guards exact method/path/origin, owned body bytes, ContentLength/TransferEncoding/framing and operation microversion headers. Changed target/body/framing or Read/Close faults stop before second transport call, close inspected Body once, and remain sticky after a repairing native retry; valid body is replaced by owned bytes. Source/context/callback causes remain reachable.
  Tests: `blockstorage/backup_import_transport_test.go:TestImportVolumeBackupPhysicalRedirectCannotChangeBodyFramingOrScope`.
- API.ImportBackup and Connection.ImportVolumeBackup expose the same owned result while preserving their actual outer operation labels. Existing native API.Import remains independent: []byte record is base64 JSON, status201 only, and typed ID/Name DTO. New import retains literal string and admits original sub400 response to its full disconnected model; native Import is neither replaced nor classified by this Proxy review.
  Tests: `blockstorage/backup_import_service_test.go:TestBackupProxyImportUsesFullDisconnectedModelWhileNativeImportRetainsBase64AndTypedResult`, `blockstorage/backup_import_service_test.go:TestBackupProxyImportPreflightKeepsCorrectOuterOperationAndCauses`.

Python/Go differences:

- The pinned Backup.import_record overwrites caller url with backups/export_record and posts that value both as route and backup_url. Go deliberately repairs both to backups/import_record and the original literal locator; exact pinned Python wire parity is not claimed.
- Two explicit Go UTF-8 strings replace Python dynamically typed service/url. Empty/control/Unicode strings are admitted, while null, numeric and container arguments and Python mutable Resource/session/constructor identities are outside this typed contract. No automatic native bytes/base64 adapter is added.
- Source constructs cls() disconnected. Go returns independent nullable24 Value plus untyped BackupID, actual raw Backup and separately owned Discovery/Applied proof rather than the same mutable Resource; location is null without CurrentLocation, scope lookup or ID seed.
- Selected nonempty microversion is forwarded literally. Current upstream Keystoneauth Session normalization of prefixes/padding/tuple/latest and complete Adapter/discovery cache, link, endpoint, envelope precedence, legacy-header or deployment-specific HTTP behavior are not replicated. Upstream references are not evidence of running the exact installed dependency.
- The discovery parser maps utility-stage maximum-before-optional-minimum and tuple order/cap. Finite body-only candidate/shapes/status policy, Unicode16 decimal table and arbitrary-precision integer strings have explicit Go boundaries; CPython runtime digit limits and dependency endpoint-data exception order are not claimed. Accepted malformed discovery is terminal with actual proof, unlike paths that catch Keystoneauth DiscoveryFailure and return None.
- Known-only descriptor mapping retains raw wire members independently. Exact JSON number/truthiness, first insertion position for duplicate alias consumption, UTF-8/surrogate/depth/backend boundaries and dictionary identity differ from Python selected JSON/Requests runtime. Ordinary malformed UTF-8 JSON tolerance is bounded; valid wrong shape or descriptor failures remain errors with consumed ID and proof.
- Owned exchange admits finite HTTP100..399 and preserves native original-policy >=400 errors. Accepted bytes/header/status and Read/Close/source/context causes are independently observed without wait/rollback/replay. Native rejected POST IO discard remains a bounded limitation; only rejected discovery404/405 explicitly observes Read/Close for safe fallback.
- Fixed source/target/body/framing and microversion ownership constrain mutable native retry/redirect hooks while live authentication and bounded caller retry remain. This review covers only the actual pinned v3 Proxy.import_backup declaration; native/v2/Resource/inherited declarations and the full SDK remain independent. Source bindings and offline HTTP tests are not Python SDK execution or authenticated OpenStack integration.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName := os.Getenv("CLOUD_NAME")
    if cloudName == "" {
        return fmt.Errorf("CLOUD_NAME is required")
    }
    backupService, servicePresent := os.LookupEnv("BACKUP_SERVICE")
    backupURL, urlPresent := os.LookupEnv("BACKUP_URL")
    if !servicePresent || !urlPresent {
        return fmt.Errorf("BACKUP_SERVICE and BACKUP_URL must be defined; empty values are allowed")
    }
    path := os.Getenv("BACKUP_IMPORT_PATH")
    if path == "" {
        path = "connection"
    }
    switch path {
    case "connection", "direct", "service":
    default:
        return fmt.Errorf("unknown BACKUP_IMPORT_PATH %q", path)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil {
        return err
    }
    input := blockstorage.ImportVolumeBackupRequest{
        BackupService: backupService,
        BackupURL: backupURL,
    }
    var result *blockstorage.ImportVolumeBackupResult
    if path == "connection" {
        result, err = conn.ImportVolumeBackup(ctx, input)
    } else {
        cinder, selectErr := conn.BlockStorageV3(ctx)
        if selectErr != nil {
            return selectErr
        }
        if path == "direct" {
            result, err = blockstorage.ImportVolumeBackup(ctx, cinder.RawClient(), input)
        } else {
            result, err = cinder.Backups.ImportBackup(ctx, backupService, backupURL)
        }
    }
    inspect(result, err)
    return err
}

func inspect(result *blockstorage.ImportVolumeBackupResult, err error) {
    if result != nil {
        fmt.Printf("logical backup ID JSON: %s\n", result.BackupID)
        fmt.Printf("import microversion: %q\n", result.Microversion)
        for i, page := range result.Discovery {
            if page != nil {
                fmt.Println("admitted discovery response:", i, page.StatusCode, len(page.Body))
            }
        }
        if result.Applied != nil {
            fmt.Println("admitted import response:", result.Applied.StatusCode, len(result.Applied.Body))
        }
        if result.Backup != nil {
            actual := result.Backup.Clone()
            fmt.Println("actual selected fields:", len(actual.Body))
            fmt.Printf("actual response ID JSON: %s\n", actual.Body["id"])
        }
    }
    if err != nil {
        fmt.Println("import error:", err)
        if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("parent context interrupted import")
        }
        var physical *resource.ResponseError
        if errors.As(err, &physical) {
            fmt.Println("current admitted error response:", physical.StatusCode, len(physical.Body))
        }
        return
    }
    if result != nil && result.Value != nil {
        pretty, formatErr := json.MarshalIndent(result.Value, "", "  ")
        if formatErr == nil {
            fmt.Println("nullable Backup:", string(pretty))
        }
    }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
backup = conn.block_storage.import_backup(
    service="cinder.backup.drivers.swift.SwiftBackupDriver",
    url="opaque-record",
)
print(backup.id, backup.to_dict())
```

## References

- [`Proxy.import_backup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1802)
- [`Backup.import_record`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L146)
- [`Resource._get_microversion`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418)
- [`maximum_supported_microversion`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L358)
- [native Import](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/backups/requests.go#L306)
- [Volume Backups](volume-backups.md)
- [Backup export record](volume-backup-export-record.md)
- [Backup restore/reset](volume-backup-actions.md)
