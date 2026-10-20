# Backup Metadata Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/fetch_backup_metadata`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.PythonMetadataProfile.FetchMetadata`

- Fetch creates a resource from an explicit ID without a preliminary member GET, GETs the metadata child route, accepts the Source final200..399 status family, parses one response object and observes only its exact metadata JSON field; absence/null/large integers/extension JSON are independent from physical proof.
  Tests: `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileFetchAndLegacyAliasKeepObservedJSON`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileFetchOmissionNullAndDecodeFailures`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges`.

Python/Go differences:

- Compatibility is explicitly selected by API.PythonMetadata(); these child routes are absent from stock Cinder27.0.0 router c36f40684e140a1492b00dd4812e8dce17fb2ebf. A physical stock404 remains an error and does not trigger fallback to ordinary backup GET/PUT.
- Go accepts an explicit validated ID string and returns owned physical evidence instead of a mutable Python Backup Resource; it neither mutates a native Backup cache nor synthesizes observed response metadata from submitted values.
- Go uses the configured client microversion and ordinary library-owned header options rather than Python session negotiation; Source MetadataMixin adds no per-request3.43 gate.
- Fetch requires an UTF-8 JSON object root and stores metadata as owned rawJSON: omission is nil and explicitnull is nonnil null; arbitrary metadata shapes and duplicate literals are retained without float conversion.

Unresolved boundaries:

- Stock Cinder27.0.0 registers no /backups/{id}/metadata child routes. The explicit PythonMetadata compatibility profile reproduces the pinned SDK request/error behavior, including physical404; deployment extension capability remains unverified and is not counted as stock SDK support.

### `python:block_storage/v3/get_backup_metadata`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.PythonMetadataProfile.GetMetadata`

- Deprecated Source alias delegates to fetch; the explicit Go profile emits the same GET and retains operation identity GetMetadata.
  Tests: `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileFetchAndLegacyAliasKeepObservedJSON`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges`.

Python/Go differences:

- Compatibility is explicitly selected by API.PythonMetadata(); these child routes are absent from stock Cinder27.0.0 router c36f40684e140a1492b00dd4812e8dce17fb2ebf. A physical stock404 remains an error and does not trigger fallback to ordinary backup GET/PUT.
- Go accepts an explicit validated ID string and returns owned physical evidence instead of a mutable Python Backup Resource; it neither mutates a native Backup cache nor synthesizes observed response metadata from submitted values.
- Go uses the configured client microversion and ordinary library-owned header options rather than Python session negotiation; Source MetadataMixin adds no per-request3.43 gate.
- Fetch requires an UTF-8 JSON object root and stores metadata as owned rawJSON: omission is nil and explicitnull is nonnil null; arbitrary metadata shapes and duplicate literals are retained without float conversion.
- Deprecated GoDoc replaces the Python RemovedInSDK50Warning; no runtime warning is emitted.

Unresolved boundaries:

- Stock Cinder27.0.0 registers no /backups/{id}/metadata child routes. The explicit PythonMetadata compatibility profile reproduces the pinned SDK request/error behavior, including physical404; deployment extension capability remains unverified and is not counted as stock SDK support.

### `python:block_storage/v3/set_backup_metadata`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.PythonMetadataProfile.SetMetadata`

- Source MetadataMixin POSTs metadata envelope including explicit{} for nil/empty input, adds no3.43 microversion gate and does not JSON-decode the response; Go owns input before callbacks and preserves opaque accepted-response evidence.
  Tests: `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSetOwnsInputAndOpaqueAcknowledgement`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges`.

Python/Go differences:

- Compatibility is explicitly selected by API.PythonMetadata(); these child routes are absent from stock Cinder27.0.0 router c36f40684e140a1492b00dd4812e8dce17fb2ebf. A physical stock404 remains an error and does not trigger fallback to ordinary backup GET/PUT.
- Go accepts an explicit validated ID string and returns owned physical evidence instead of a mutable Python Backup Resource; it neither mutates a native Backup cache nor synthesizes observed response metadata from submitted values.
- Go uses the configured client microversion and ordinary library-owned header options rather than Python session negotiation; Source MetadataMixin adds no per-request3.43 gate.
- Go uses map[string]string for the documented string kwargs; opaque response is returned without manufacturing the Python input-based mutable cache.

Unresolved boundaries:

- Stock Cinder27.0.0 registers no /backups/{id}/metadata child routes. The explicit PythonMetadata compatibility profile reproduces the pinned SDK request/error behavior, including physical404; deployment extension capability remains unverified and is not counted as stock SDK support.

### `python:block_storage/v3/delete_backup_metadata`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.PythonMetadataProfile.DeleteMetadata`

- Source keys=None PUTs metadata:{}, nonnil empty keys issue noHTTP, and other keys DELETE sequentially preserving duplicates until first error. Go retains ordered prior acknowledgements, captures literal key segments and never falls back to ordinary backup member GET/PUT.
  Tests: `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileDeleteNilEmptyLiteralKeysAndPartialFailure`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges`.

Python/Go differences:

- Compatibility is explicitly selected by API.PythonMetadata(); these child routes are absent from stock Cinder27.0.0 router c36f40684e140a1492b00dd4812e8dce17fb2ebf. A physical stock404 remains an error and does not trigger fallback to ordinary backup GET/PUT.
- Go accepts an explicit validated ID string and returns owned physical evidence instead of a mutable Python Backup Resource; it neither mutates a native Backup cache nor synthesizes observed response metadata from submitted values.
- Go uses the configured client microversion and ordinary library-owned header options rather than Python session negotiation; Source MetadataMixin adds no per-request3.43 gate.
- Go validates and escapes literal keys once; it returns partial HTTP acknowledgements rather than None and does not reproduce mutable-resource cache removal KeyError behavior.

Unresolved boundaries:

- Stock Cinder27.0.0 registers no /backups/{id}/metadata child routes. The explicit PythonMetadata compatibility profile reproduces the pinned SDK request/error behavior, including physical404; deployment extension capability remains unverified and is not counted as stock SDK support.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
    "github.com/gophercloud/gophercloud/v2"
)

func fetchBackupMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]string, error) {
    profile := backups.New(client).PythonMetadata()
    result, err := profile.FetchMetadata(ctx, id,
        backups.WithMetadataHeader("X-Request-Id", "application-request"))
    if err != nil {
        return nil, err
    }
    var values map[string]string
    if result.Metadata != nil {
        if err := json.Unmarshal(result.Metadata, &values); err != nil {
            return nil, err
        }
    }
    return values, nil
}
```

## References

- [v3 router](https://github.com/openstack/cinder/blob/c36f40684e140a1492b00dd4812e8dce17fb2ebf/cinder/api/v3/router.py#L183-L187)
- [Cinder Metadata Server Contracts](../../docs/cinder-metadata-server-contracts.md)
- [Backup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/backup.py#L27)
- [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1772-L1867)
- [metadata_python_test.go](backups/metadata_python_test.go)
- [metadata.go](backups/metadata.go)
