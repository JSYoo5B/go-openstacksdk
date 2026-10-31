# Mutations Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/create_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordCreateHeaders`

- One bodyless literal-route PUT accepts only actual201/202, returns opaque ContainerResponse, and performs no lookup/HEAD/implicit refresh.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreWireIdentity`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsWireAndIdentity`, `connectiontest/connection_objectstorage_container_lifecycle_test.go:TestConnectionContainerLifecycleSharesClientAndPreservesNative`.
- Required safe literal UTF8 container and captured effective reverse-prefix source/provider/context/header validation precede HTTP; native empty/slash and unsupported causes remain visible.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreCompletePreflight`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsInvalidInputs`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsPreflightAndOptions`.
- Concrete functional options snapshot Header/Metadata maps before callbacks, initialize maps, invoke each callback once, clone retained storage, replace full configuration/metadata and apply canonical header last overwrite.
  Tests: `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsSnapshotsAndOverwrite`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsReplacementAndDefaults`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsParallelReuse`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsPreflightAndOptions`.
- Suffix-only ASCII metadata keys and literal legalUTF8 values produce exact X-Container-Meta headers; nil/empty maps still issue PUT, source/caller input remains owned. Omitted keys and newer emptyvalue deletion follow pinnedserver merge, with no clientRMW or inferred state.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreWireIdentity`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsInvalidInputs`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsWireAndIdentity`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsPreflightAndOptions`.
- Actual accepted201/202 copied bytes/header/status survive read/Close/context/source failure with ResponseError and original causes; body closes once and accepted handling failure is never replayed.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreAcceptedResponseFailures`, `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreSourceAndContext`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsAcknowledgementOwnership`.
- Live original-provider auth, native prebody retry/reauth/fixedscope redirect policy and explicit advanced header policy remain bounded; new204rejection does not alter existing generated/native Create201/202/204 ABI.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreNativePolicy`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsSourceAndNativePolicy`, `connectiontest/connection_objectstorage_container_lifecycle_test.go:TestConnectionContainerLifecycleSharesClientAndPreservesNative`, `internal/cmd/sdkgen/swift_container_lifecycle_test.go:TestSwiftContainerLifecyclePreservesNativeAndRegistry`.
- CreateRecord sends one literal bodyless PUT with Source declared Body/Header attrs and constructor metadata prefix semantics. Scalar Python-string spellings, string-list join, false/null/empty values, system/wire Header aliases and metadata overlay are owned; unknown attrs/id are ignored and duplicate required name/ambiguous map aliases are rejected before HTTP. Semantic descriptor names are case-sensitive; actual wire Header names remain case-insensitive. Source constructor typed seed descriptors are validated before option callbacks/HTTP; raw input seeds and actual Header spellings are retained unchanged.
  Tests: `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordCreatesHeadersAndDeclaredProjection`, `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordSourceCaptureSnapshotAndOptions`, `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordSeedDescriptorsFailBeforeOptionsAndHTTP`.
- Owned ContainerRecord separates input SeedBody/SeedHeaders from complete actual Wire/Envelope/Header/StatusCode and the declared Resource view. Source descriptor int/bool and count/header aliases preserve omitted versus null, seed overlay and unknown wire fields. Opaque malformed/empty response body keeps seeds; valid scalar/list/null body errors retain physical response. Custom response metadata excludes known consumed Header fields and never invents input as observed metadata.
  Tests: `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordCreatesHeadersAndDeclaredProjection`, `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordOpaqueSuccessRetainsSeedsAndAliasDefaults`, `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordPublicFixtureAndIndependentDefaults`.
- Source generic <400 policy accepts actual HTTP100..399, including204/299/399, with no HEAD/lookup/implicit refresh. Captured source precedes custom JSON/options; nested attrs and ordinary Header factory inputs are snapshots. Physical accepted/rejected read/Close/cancel/sticky source faults retain original causes and actual receipts; native clean retry is preserved but request body/owned attr headers cannot change.
  Tests: `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordGenericStatusesAndCleanMissingPolicy`, `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordPhysicalFaultsPreserveReceiptsAndDirty404`, `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordSourceCaptureSnapshotAndOptions`, `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordRetryRetainsTypedHeadersAndOriginalCauses`.

Python/Go differences:

- The new callable implements the Source name-string named workflow. Existing generated/native methods and strict CreateContainer/DeleteContainer retain their separate contracts and counts. Go owns projections/receipts instead of mutable Resource mutation; R1/C1/S1 Resource cache/dirty/Adapter/session/location behavior remains separately tracked.
- Go rejects ambiguous attribute/metadata/header aliases rather than depending on Python insertion order lost by maps/http.Header. JSON-domain Python spellings use stable Go JSON dict order. Metadata storage is per invocation instead of reproducing pinned class-level cross-instance leakage.
- ACK/ignored404 does not prove durable cluster state, historical existence, global absence, replication or successful server ACL/quota/policy. These deployment properties and authenticated acceptance remain separate SDK release gates.

### `python:object_store/v1/delete_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordDeleteHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordDeleteIgnoreMissing`

- One bodyless DELETE uses the literal container route without lookup, accountHEAD, emptiness probe, objectpurge, wait or cleanup; actual204 returns owned opaque response.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreWireIdentity`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsWireAndIdentity`, `connectiontest/connection_objectstorage_container_lifecycle_test.go:TestConnectionContainerLifecycleSharesClientAndPreservesNative`.
- IgnoreMissing nil/true accepts physically owned404 solely for observation; only clean read/Close/context/source checks return actual404 ContainerResponse with IgnoredMissingtrue and nilerror. It is not proof of historical existence or globalabsence and bypasses native error RetryFunc.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreStrictStatusesAndMissing`, `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreNativePolicy`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsReplacementAndDefaults`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsWireAndIdentity`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsMissingAndFailures`.
- Explicitfalse accepts only204;404 yields nilresult/nativeunexpectederror. Nested/transport404,401/403/409/202 and handling/source/context errors are never suppressed.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreStrictStatusesAndMissing`, `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreNativePolicy`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsMissingAndFailures`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsAcknowledgementOwnership`.
- Owned Header maps and IgnoreMissing pointer snapshots support callback-retention separation, fullreplacement, canonical header lastwins, casealias rejection and parallel helper reuse without source mutation.
  Tests: `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsReplacementAndDefaults`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsParallelReuse`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsPreflightAndOptions`.
- Required safe literalUTF8 name and captured effective ResourceBase/source/provider/options/context validation precede DELETE; no name resolver or URLdecode is performed.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreCompletePreflight`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsInvalidInputs`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsPreflightAndOptions`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsWireAndIdentity`.
- Accepted204 and allowed404 actualBody/Header/StatusCode survive read/Close/context/source failure with original/customcontextcauses and wholeResponseError; dirty404 has IgnoredMissingfalse, oneClose and no acceptedresponse replay.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreAcceptedResponseFailures`, `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreSourceAndContext`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsAcknowledgementOwnership`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsMissingAndFailures`.
- Servernonempty409 and accountmissing404 remain bounded; liveauth/native prebody hooks and typedreauth evidence are retained. Existing generated/native Delete202/204 and metadata/Resources ABI remain unchanged.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreNativePolicy`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsSourceAndNativePolicy`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsMissingAndFailures`, `connectiontest/connection_objectstorage_container_lifecycle_test.go:TestConnectionContainerLifecycleSharesClientAndPreservesNative`, `internal/cmd/sdkgen/swift_container_lifecycle_test.go:TestSwiftContainerLifecyclePreservesNativeAndRegistry`.
- DeleteRecord sends one bodyless literal container DELETE, accepts generic HTTP100..399 and has Source ignore_missing=True default.404 remains a native rejection eligible for the existing RetryFunc; only an ultimately clean404 is suppressed and returned as actual ignored receipt. Explicitfalse, server403/409, read/Close/context/source-mixed404 preserve original causes; no HEAD/purge/existence/wait/cleanup is added.
  Tests: `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordGenericStatusesAndCleanMissingPolicy`, `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordPhysicalFaultsPreserveReceiptsAndDirty404`.

Python/Go differences:

- The new callable implements the Source name-string named workflow. Existing generated/native methods and strict CreateContainer/DeleteContainer retain their separate contracts and counts. Go owns projections/receipts instead of mutable Resource mutation; R1/C1/S1 Resource cache/dirty/Adapter/session/location behavior remains separately tracked.
- Go rejects ambiguous attribute/metadata/header aliases rather than depending on Python insertion order lost by maps/http.Header. JSON-domain Python spellings use stable Go JSON dict order. Metadata storage is per invocation instead of reproducing pinned class-level cross-instance leakage.
- ACK/ignored404 does not prove durable cluster state, historical existence, global absence, replication or successful server ACL/quota/policy. These deployment properties and authenticated acceptance remain separate SDK release gates.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
container = conn.object_store.create_container(
    "assets", read_ACL=".r:*", is_content_type_detected=False,
    history_location="", metadata={"owner": "team", "disabled": False},
)
print(container.name, container.metadata)
conn.object_store.delete_container("assets")
conn.object_store.delete_container("assets", ignore_missing=False)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    swift, err := conn.ObjectStorageV1(ctx)
    if err != nil { log.Fatal(err) }
    created, err := swift.Containers.CreateRecord(ctx, "assets", map[string]any{
        "read_ACL": ".r:*",
        "is_content_type_detected": false,
        "history_location": "",
        "metadata": map[string]any{"owner": "team", "disabled": false},
    }, containers.WithRecordCreateHeader("X-Client-Request-Id", "create-assets"))
    if err != nil { log.Fatal(err) }
    fmt.Printf("PUT status=%d name=%s observed_metadata=%v\n",
        created.StatusCode, created.Resource.Body["name"], created.Metadata)

    deleted, err := swift.Containers.DeleteRecord(ctx, "assets")
    if err != nil { log.Fatal(err) }
    fmt.Printf("DELETE status=%d ignored_missing=%t\n", deleted.StatusCode, deleted.IgnoredMissing)

    // To reproduce ignore_missing=False, select the concrete option explicitly.
    _, err = swift.Containers.DeleteRecord(ctx, "assets",
        containers.WithRecordDeleteIgnoreMissing(false))
    if err != nil { fmt.Println("strict deletion:", err) }
}
```

## References

- [Python Record Test](python_record_test.go)
