# Federation

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_mapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.CreateSourceMappingRecord`

- CreateMapping sends PUT OS-FEDERATION/mappings/{id} with id and schema_version in the mapping envelope via WithCreateMappingField, as Python does, but the decoded Mapping drops schema_version and unknown rule keys.
  Tests: `identity/v3/federation/python_parity_test.go:TestPythonMappingRequestsAndDecodeGaps`.
- Owned member PUT preserves arbitrary raw rule keys/schema writes, required wire ID and ignored constructor name and nullable/list/string Resource views; actual response overrides seed and retains independent Wire evidence.
  Tests: `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingCreatePreservesRawRulesAndNullableDescriptorView`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingCreateIgnoresNameDefaultsAndStatusPolicy`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingOwnershipAndCallbackGuards`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingPreflightCollisionsAndInvalidBodies`.

Python/Go differences:

- The additive SourceMappingRecord API preserves the existing native and MappingRecord APIs. It owns finite JSON attributes and safe string route IDs; arbitrary Python objects and mutable Resource dirty-baseline/recommit state are represented by a fresh Go operation instead.
- Raw mutation base_path overrides are explicitly unsupported; headers and microversion options are Go conveniences.
- Python find_mapping deprecation warnings are not emitted. full Requests codec, runtime/session, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally.

### `python:identity/v3/find_mapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.FindSourceMappingRecord`

- GET-first lookup falls back only on clean 400/403/404, scans all pages without a name query, handles missing/strict/duplicate results and preserves physical terminal failures.
  Tests: `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingFindGETFirstFallbackPaginationAndDuplicate`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingPhysicalFaultsPreserveProofAndStopFallback`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingOwnershipAndCallbackGuards`.

Python/Go differences:

- The additive SourceMappingRecord API preserves the existing native and MappingRecord APIs. It owns finite JSON attributes and safe string route IDs; arbitrary Python objects and mutable Resource dirty-baseline/recommit state are represented by a fresh Go operation instead.
- Raw mutation base_path overrides are explicitly unsupported; headers and microversion options are Go conveniences.
- Python find_mapping deprecation warnings are not emitted. full Requests codec, runtime/session, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally.

### `python:identity/v3/update_mapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.UpdateSourceMappingRecord`

- UpdateMapping sends PATCH OS-FEDERATION/mappings/{id} and always includes rules, as null when empty, so a schema_version-only update differs from Python's body.
  Tests: `identity/v3/federation/python_parity_test.go:TestPythonMappingRequestsAndDecodeGaps`.
- Fresh scalar-ID PATCH includes only supplied dirty raw fields; omitted rules are absent, explicit null is sent, raw name does not change the route, id collision is rejected and empty/unknown-only updates do not send HTTP.
  Tests: `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingUpdateDirtyNoopNullAndIgnoredName`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingPreflightCollisionsAndInvalidBodies`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingOwnershipAndCallbackGuards`.

Python/Go differences:

- The additive SourceMappingRecord API preserves the existing native and MappingRecord APIs. It owns finite JSON attributes and safe string route IDs; arbitrary Python objects and mutable Resource dirty-baseline/recommit state are represented by a fresh Go operation instead.
- Raw mutation base_path overrides are explicitly unsupported; headers and microversion options are Go conveniences.
- Python find_mapping deprecation warnings are not emitted. full Requests codec, runtime/session, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally.

## References

- [Python Source Mapping](python-source-mapping.md)
- [Python Mapping Records](python-mapping-records.md)
