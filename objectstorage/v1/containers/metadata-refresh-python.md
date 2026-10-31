# Metadata Refresh Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/set_container_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.SetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.SetMetadataAndRefresh`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataRefresh`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataRefreshOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataRefreshHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataRefreshHeaders`

- Set is one bodyless POST with exactly one X-Container-Meta prefix and strict actual204, returning opaque acknowledgement without automatic HEAD, submitted-state projection, container creation or clear-all. Nil/empty maps still POST and empty values remove only the specified keys.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreRoutesAndEvidence`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`, `connectiontest/connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
- The caller map is snapshotted before callbacks. Suffix-only nonempty original ASCII token keys reject full prefixes and case aliases; literal UTF8 field values preserve spaces/percent/Unicode/HTAB/empty without str coercion, trim or URL encoding, and invalid input stops before HTTP.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreCompletePreflight`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsFactorySnapshots`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`.
- Exact ordinary system wire headers for ACL/sync/version/history/content type may be provided or explicitly empty. Python read_ACL/sync_key aliases are not translated; custom suffix inputs remain custom metadata. Auth/hop/body/Newest/custom-prefix ownership prevents generic-header interference while advanced native retry headers remain native policy.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreRoutesAndEvidence`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsSourceAndNativePolicy`.
- Mutation full opts wholly replaces headers, singular/plural helpers merge with alias validation, factory/callback handles own their storage and options support parallel reuse. Literal-name/source/target/header/context preflight and joined callback/source/custom cancellation causes are preserved.
  Tests: `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsFactorySnapshots`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsReplacementAndNewest`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreCompletePreflight`.
- Accepted204 Set acknowledgement owns opaque Body/Header/Status and preserves repeated or malformed metadata-looking headers without projection. Read/Close/context/custom/source failure returns result plus whole ResponseError, closes once and never replays an accepted response.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreAcceptedFailures`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
- Set produces no result on missing404, unexpected200/201/202 or native/transport errors and preserves original typed evidence. Captured effectivebase/raw source/liveauth and request/body/status guards retain native prebody retry/reauth/redirect policy and legacy Update acceptance without performing a refresh HEAD.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsSourceAndNativePolicy`, `connectiontest/connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
- Default refresh=true POST then HEAD and explicit false is callable through WithMetadataRefresh(false); later scalar wins and whole concrete opts resets default.
  Tests: `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataSetterRefreshesObservedStateAndSourceValues`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataRefreshFalseLaterOptionsAndReusableSnapshots`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataEverySystemAliasAndPrefixedConvention`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataRefreshFailuresKeepAcknowledgementWithoutReplay`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataInputsCallbacksAndConcurrentReuseAreOwned`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataPreflightAndSourceStatusPolicy`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataCapturesSourceBeforeCustomJSONMarshaler`.
- All eight Source canonical and exact wire system aliases, full custom prefix, JSON Python string conversion and physical per-phase evidence.
  Tests: `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataSetterRefreshesObservedStateAndSourceValues`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataRefreshFalseLaterOptionsAndReusableSnapshots`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataEverySystemAliasAndPrefixedConvention`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataRefreshFailuresKeepAcknowledgementWithoutReplay`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataInputsCallbacksAndConcurrentReuseAreOwned`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataPreflightAndSourceStatusPolicy`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataCapturesSourceBeforeCustomJSONMarshaler`.

Python/Go differences:

- Go returns independently owned physical response/phase proof and accepts context/explicit route names; Python None/same mutable Resource return is part of separate Resource scope.
- Metadata values cover JSON-domain Python string/truthiness conversion; arbitrary __str__, nonJSON objects, float repr and Unicode runtime identity are bounded; colliding incoming Go map wire aliases fail before HTTP.
- Go metadata token/field-value/single-header validation and native per-request transport policy remain explicit boundaries.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: R1/C1/S1 mutable Resource descriptor/cache and Adapter/session behavior remain independently tracked.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Server ACL/quota/encoding/version/SLO/symlink and durable replication are deployment contracts; read/merge/write offers no CAS/rollback/whole-operation replay.

### `python:object_store/v1/delete_container_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.DeleteMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.DeleteMetadataKeys`

- DeleteMetadata sends one bodyless POST carrying empty X-Container-Meta values, never HTTP DELETE or X-Remove syntax. Nil/empty keys still POST, do not clear all metadata, and do not trigger refresh or fallback; only actual204 creates acknowledgement.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreRoutesAndEvidence`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`, `connectiontest/connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
- Delete copies the key slice before callbacks. Suffix-only nonempty original ASCII token keys reject full prefixes, duplicates and case aliases; required literal container/source/header/context validation fails before HTTP with original typed/callback/custom context causes.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreCompletePreflight`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsFactorySnapshots`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`.
- Delete shares concrete whole-replacement mutation opts and canonical singular/plural merging, independently copied callback/factory storage and concurrent reuse. Exact ordinary system wire headers may carry empty values; Python system alias mapping is not inferred from a custom key slice.
  Tests: `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsReplacementAndNewest`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`.
- Actual204 Delete raw Body/Header/Status and joined read/Close/context/custom/source failures are preserved in independent acknowledgement and whole ResponseError with one Close/no resend. Mutation headers are passive and do not prove refreshed metadata state.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreAcceptedFailures`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`, `connectiontest/connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`.
- Delete preserves missing404 and other unexpected/native failures with nil result and original typed error; no IgnoreMissing or container deletion is inferred. Captured effectivebase/raw source/live original auth and native prebody/reauth/fixed-scope redirect policy remain guarded for target/body/status with advanced native MoreHeaders retained.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsSourceAndNativePolicy`, `connectiontest/connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
- One bodyless POST including nil/empty keys; duplicate keys collapse; canonical/wire system aliases and full custom prefix reach empty header values.
  Tests: `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataEmptyAndPrefixedDeleteFollowSourcePOST`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataEverySystemAliasAndPrefixedConvention`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataRefreshFailuresKeepAcknowledgementWithoutReplay`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataInputsCallbacksAndConcurrentReuseAreOwned`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataPreflightAndSourceStatusPolicy`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataCapturesSourceBeforeCustomJSONMarshaler`.

Python/Go differences:

- Go returns independently owned physical response/phase proof and accepts context/explicit route names; Python None/same mutable Resource return is part of separate Resource scope.
- Metadata values cover JSON-domain Python string/truthiness conversion; arbitrary __str__, nonJSON objects, float repr and Unicode runtime identity are bounded; colliding incoming Go map wire aliases fail before HTTP.
- Go metadata token/field-value/single-header validation and native per-request transport policy remain explicit boundaries.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: R1/C1/S1 mutable Resource descriptor/cache and Adapter/session behavior remain independently tracked.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Server ACL/quota/encoding/version/SLO/symlink and durable replication are deployment contracts; read/merge/write offers no CAS/rollback/whole-operation replay.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := containers.New(swiftClient)
result, err := api.SetMetadataAndRefresh(ctx, "container", map[string]any{
    "owner": "ops",
    "enabled": false,
    "X-Container-Meta-External": "value",
})
if err != nil {
    return err
}
observed := result.Refreshed.Metadata.Values["owner"]
_ = observed
```

## References

- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L210-L251)
- [BaseResource.set_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L73-L92)
- [Container._system_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/container.py#L22-L34)
