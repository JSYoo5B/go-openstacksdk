# Containers

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/get_container_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.GetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithGetMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithGetMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithGetMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithGetMetadataNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithoutGetMetadataNewest`

- One bodyless HEAD targets the once-escaped literal container on the captured effective ResourceBase/Endpoint, including escaped reverse prefix and trailing-slash policy; strict actual204 and no discovery, Name lookup, parent HEAD or fallback.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreRoutesAndEvidence`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`, `connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`.
- Newest nil omits X-Newest, explicit false/true is sent only on HEAD and Without clears. Whole opts replaces configuration; ordinary singular/plural headers merge canonically with last overwrite while incoming and final case aliases are rejected.
  Tests: `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsReplacementAndNewest`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`.
- Factories snapshot maps/pointers and every callback receives independently owned initialized storage. Retained callback handles and concurrent reusable options do not alter another call, and option/source/context causes are preserved.
  Tests: `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsFactorySnapshots`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`.
- Get atomically projects a nonnil lowercased custom suffix map including Temp URL keys, nullable exact signed64 counters and literal Timestamp/LastModified. Original nonASCII keys/suffixes, alias/duplicate/multivalue headers and malformed counters reject without partial Metadata while raw headers remain intact.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreAtomicHeaderProjection`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsAtomicHeaders`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`, `connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`.
- Required literal name preserves native empty/slash typed errors and adds UTF8/control/backslash/dot safety without trimming or deployment length inference. Nil context/API/client/provider, wrong Type, invalid Endpoint/effectivebase/header, reserved headers, nil callbacks and canceled contexts stop before HTTP.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreCompletePreflight`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`.
- Only actual204 creates owned Get Body/Header/Status. Accepted read/Close/context/custom cause/source/projection failures retain the result with nil Metadata and whole independent ResponseError; one Close/no accepted replay and original strconv cause are retained.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreAcceptedFailures`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreAtomicHeaderProjection`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsAtomicHeaders`, `connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`.
- Original source/provider/raw Endpoint/ResourceBase/Type/Microversion/target and ordinary headers are captured before callbacks and rechecked; live original auth, native prebody retry/reauth fields and configured fixed-scope redirects remain. Advanced RetryFunc.MoreHeaders and native Get200/204, Update201/202/204 ABI remain native policy.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsSourceAndNativePolicy`, `connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`.

Python/Go differences:

- Python returns a mutable Container through Resource/header descriptors and Adapter/session state; Go owns raw actual204 evidence and atomic nullable MetadataInfo.
- Values intentionally includes Temp-URL-Key and Temp-URL-Key-2; literal LastModified/Timestamp and signed nullable counters differ from Python known-descriptor consumption and native float/date/ACL parsing.
- Python accepts a literal name or mutable Container Resource; Go takes a required literal UTF-8 name with no lookup or parent HEAD. It preserves a once-escaped child route on configured ResourceBase/Endpoint and rejects empty/slash/control/backslash/dot unsafe names rather than adapting arbitrary Resource state.
- Python header translation accepts statuses below400. Go accepts actual204 only and validates canonical metadata/counters/timestamp headers atomically; raw Body/Header/Status and original typed native/transport or accepted-response causes remain available on the documented paths.
- Python known ACL/sync/version/storage/content-type/header descriptors and mutable _last_headers are represented by owned raw Header in Go, rather than additional typed descriptor aliases. Missing typed counters stay nil, values use exact signed int64, and timestamp/LastModified stay literal. Temp URL keys deliberately remain in Values. Cache/session/header descriptor coercion and automatic setter refresh are separate API scope; deployment authorization and limits are server policy.

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
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreRoutesAndEvidence`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`, `connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
- The caller map is snapshotted before callbacks. Suffix-only nonempty original ASCII token keys reject full prefixes and case aliases; literal UTF8 field values preserve spaces/percent/Unicode/HTAB/empty without str coercion, trim or URL encoding, and invalid input stops before HTTP.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreCompletePreflight`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsFactorySnapshots`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`.
- Exact ordinary system wire headers for ACL/sync/version/history/content type may be provided or explicitly empty. Python read_ACL/sync_key aliases are not translated; custom suffix inputs remain custom metadata. Auth/hop/body/Newest/custom-prefix ownership prevents generic-header interference while advanced native retry headers remain native policy.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreRoutesAndEvidence`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsSourceAndNativePolicy`.
- Mutation full opts wholly replaces headers, singular/plural helpers merge with alias validation, factory/callback handles own their storage and options support parallel reuse. Literal-name/source/target/header/context preflight and joined callback/source/custom cancellation causes are preserved.
  Tests: `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsFactorySnapshots`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsReplacementAndNewest`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreCompletePreflight`.
- Accepted204 Set acknowledgement owns opaque Body/Header/Status and preserves repeated or malformed metadata-looking headers without projection. Read/Close/context/custom/source failure returns result plus whole ResponseError, closes once and never replays an accepted response.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreAcceptedFailures`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
- Set produces no result on missing404, unexpected200/201/202 or native/transport errors and preserves original typed evidence. Captured effectivebase/raw source/liveauth and request/body/status guards retain native prebody retry/reauth/redirect policy and legacy Update acceptance without performing a refresh HEAD.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsSourceAndNativePolicy`, `connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
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
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreRoutesAndEvidence`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`, `connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
- Delete copies the key slice before callbacks. Suffix-only nonempty original ASCII token keys reject full prefixes, duplicates and case aliases; required literal container/source/header/context validation fails before HTTP with original typed/callback/custom context causes.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreCompletePreflight`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsFactorySnapshots`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`.
- Delete shares concrete whole-replacement mutation opts and canonical singular/plural merging, independently copied callback/factory storage and concurrent reuse. Exact ordinary system wire headers may carry empty values; Python system alias mapping is not inferred from a custom key slice.
  Tests: `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsReplacementAndNewest`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/containers/metadata_options_test.go:TestContainerMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsPreflightAndOptions`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsWireAndIdentity`.
- Actual204 Delete raw Body/Header/Status and joined read/Close/context/custom/source failures are preserved in independent acknowledgement and whole ResponseError with one Close/no resend. Mutation headers are passive and do not prove refreshed metadata state.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreAcceptedFailures`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`, `connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`.
- Delete preserves missing404 and other unexpected/native failures with nil result and original typed error; no IgnoreMissing or container deletion is inferred. Captured effectivebase/raw source/live original auth and native prebody/reauth/fixed-scope redirect policy remain guarded for target/body/status with advanced native MoreHeaders retained.
  Tests: `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/containers/metadata_core_test.go:TestContainerMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsSourceAndNativePolicy`, `connection_objectstorage_container_metadata_test.go:TestConnectionContainerMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_container_metadata_test.go:TestSwiftContainerMetadataPreservesNativeAndRegistry`, `objectstorage/v1/containers/metadata_contracts_test.go:TestContainerMetadataContractsMutationEvidence`.
- One bodyless POST including nil/empty keys; duplicate keys collapse; canonical/wire system aliases and full custom prefix reach empty header values.
  Tests: `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataEmptyAndPrefixedDeleteFollowSourcePOST`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataEverySystemAliasAndPrefixedConvention`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataRefreshFailuresKeepAcknowledgementWithoutReplay`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataInputsCallbacksAndConcurrentReuseAreOwned`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataPreflightAndSourceStatusPolicy`, `objectstorage/v1/containers/metadata_refresh_test.go:TestSwiftPythonContainerMetadataCapturesSourceBeforeCustomJSONMarshaler`.

Python/Go differences:

- Go returns independently owned physical response/phase proof and accepts context/explicit route names; Python None/same mutable Resource return is part of separate Resource scope.
- Metadata values cover JSON-domain Python string/truthiness conversion; arbitrary __str__, nonJSON objects, float repr and Unicode runtime identity are bounded; colliding incoming Go map wire aliases fail before HTTP.
- Go metadata token/field-value/single-header validation and native per-request transport policy remain explicit boundaries.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: R1/C1/S1 mutable Resource descriptor/cache and Adapter/session behavior remain independently tracked.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Server ACL/quota/encoding/version/SLO/symlink and durable replication are deployment contracts; read/merge/write offers no CAS/rollback/whole-operation replay.

### `python:object_store/v1/create_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithRecordCreateHeaders`

- One bodyless literal-route PUT accepts only actual201/202, returns opaque ContainerResponse, and performs no lookup/HEAD/implicit refresh.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreWireIdentity`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsWireAndIdentity`, `connection_objectstorage_container_lifecycle_test.go:TestConnectionContainerLifecycleSharesClientAndPreservesNative`.
- Required safe literal UTF8 container and captured effective reverse-prefix source/provider/context/header validation precede HTTP; native empty/slash and unsupported causes remain visible.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreCompletePreflight`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsInvalidInputs`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsPreflightAndOptions`.
- Concrete functional options snapshot Header/Metadata maps before callbacks, initialize maps, invoke each callback once, clone retained storage, replace full configuration/metadata and apply canonical header last overwrite.
  Tests: `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsSnapshotsAndOverwrite`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsReplacementAndDefaults`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsParallelReuse`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsPreflightAndOptions`.
- Suffix-only ASCII metadata keys and literal legalUTF8 values produce exact X-Container-Meta headers; nil/empty maps still issue PUT, source/caller input remains owned. Omitted keys and newer emptyvalue deletion follow pinnedserver merge, with no clientRMW or inferred state.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreWireIdentity`, `objectstorage/v1/containers/lifecycle_options_test.go:TestContainerLifecycleOptionsInvalidInputs`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsWireAndIdentity`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsPreflightAndOptions`.
- Actual accepted201/202 copied bytes/header/status survive read/Close/context/source failure with ResponseError and original causes; body closes once and accepted handling failure is never replayed.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreAcceptedResponseFailures`, `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreSourceAndContext`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsAcknowledgementOwnership`.
- Live original-provider auth, native prebody retry/reauth/fixedscope redirect policy and explicit advanced header policy remain bounded; new204rejection does not alter existing generated/native Create201/202/204 ABI.
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreNativePolicy`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsSourceAndNativePolicy`, `connection_objectstorage_container_lifecycle_test.go:TestConnectionContainerLifecycleSharesClientAndPreservesNative`, `internal/cmd/sdkgen/swift_container_lifecycle_test.go:TestSwiftContainerLifecyclePreservesNativeAndRegistry`.
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
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreWireIdentity`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsWireAndIdentity`, `connection_objectstorage_container_lifecycle_test.go:TestConnectionContainerLifecycleSharesClientAndPreservesNative`.
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
  Tests: `objectstorage/v1/containers/lifecycle_core_test.go:TestContainerLifecycleCoreNativePolicy`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsSourceAndNativePolicy`, `objectstorage/v1/containers/lifecycle_contracts_test.go:TestContainerLifecycleContractsMissingAndFailures`, `connection_objectstorage_container_lifecycle_test.go:TestConnectionContainerLifecycleSharesClientAndPreservesNative`, `internal/cmd/sdkgen/swift_container_lifecycle_test.go:TestSwiftContainerLifecyclePreservesNativeAndRegistry`.
- DeleteRecord sends one bodyless literal container DELETE, accepts generic HTTP100..399 and has Source ignore_missing=True default.404 remains a native rejection eligible for the existing RetryFunc; only an ultimately clean404 is suppressed and returned as actual ignored receipt. Explicitfalse, server403/409, read/Close/context/source-mixed404 preserve original causes; no HEAD/purge/existence/wait/cleanup is added.
  Tests: `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordGenericStatusesAndCleanMissingPolicy`, `objectstorage/v1/containers/python_record_test.go:TestPythonContainerRecordPhysicalFaultsPreserveReceiptsAndDirty404`.

Python/Go differences:

- The new callable implements the Source name-string named workflow. Existing generated/native methods and strict CreateContainer/DeleteContainer retain their separate contracts and counts. Go owns projections/receipts instead of mutable Resource mutation; R1/C1/S1 Resource cache/dirty/Adapter/session/location behavior remains separately tracked.
- Go rejects ambiguous attribute/metadata/header aliases rather than depending on Python insertion order lost by maps/http.Header. JSON-domain Python spellings use stable Go JSON dict order. Metadata storage is per invocation instead of reproducing pinned class-level cross-instance leakage.
- ACK/ignored404 does not prove durable cluster state, historical existence, global absence, replication or successful server ACL/quota/policy. These deployment properties and authenticated acceptance remain separate SDK release gates.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListContainers(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	for container, err := range service.Containers.Resources.List(ctx,
		resource.WithPageSize(100),
		resource.WithMaxItems(20),
		resource.WithPaginated(false),
	) {
		if err != nil {
			return err
		}
		fmt.Println(container.Name, container.Count, container.Bytes)
	}
	return nil
}
```

```python
container = conn.object_store.get_container_metadata("books")
print(container.metadata.get("book"))
refreshed = conn.object_store.set_container_metadata(
    container, book="Moby Dick", note="literal %2F 雪"
)
conn.object_store.delete_container_metadata(refreshed, ["note"])
```

```go
package main

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

func changeContainerMetadata(ctx context.Context, api *containers.API, name string) error {
    getOptions := []containers.GetMetadataOption{
        containers.WithGetMetadataOpts(containers.GetMetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithGetMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        containers.WithGetMetadataHeader("X-Request-Label", "container-metadata"),
        containers.WithGetMetadataNewest(false),
    }
    before, err := api.GetMetadata(ctx, name, getOptions...)
    if before != nil { fmt.Println(before.StatusCode) }
    if err != nil { return err }
    if before.Metadata.ObjectCount != nil { fmt.Println(*before.Metadata.ObjectCount) }

    mutationOptions := []containers.MetadataOption{
        containers.WithMetadataOpts(containers.MetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        containers.WithMetadataHeader("X-Request-Label", "metadata-mutation"),
    }
    acknowledgement, err := api.SetMetadata(ctx, name, map[string]string{
        "book": "Moby Dick", "note": "literal %2F 雪",
    }, mutationOptions...)
    if acknowledgement != nil { fmt.Println(acknowledgement.StatusCode) }
    if err != nil { return err }

    refreshOptions := append([]containers.GetMetadataOption(nil), getOptions...)
    refreshOptions = append(refreshOptions, containers.WithoutGetMetadataNewest())
    refreshed, err := api.GetMetadata(ctx, name, refreshOptions...)
    if err != nil { return err }
    fmt.Println(refreshed.Metadata.Values["book"])

    removed, err := api.DeleteMetadata(ctx, name, []string{"note"}, mutationOptions...)
    if removed != nil { fmt.Println(removed.StatusCode) }
    return err
}

func main() {}
```

```python
created = conn.object_store.create_container(
    "books-lifecycle", metadata={"owner": "ops", "note": "literal %2F 雪"}
)
conn.object_store.delete_container(created, ignore_missing=False)
conn.object_store.delete_container("books-lifecycle")
```

```go
package main

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

func containerLifecycle(ctx context.Context, api *containers.API, name string) error {
    createOptions := []containers.CreateContainerOption{
        containers.WithCreateContainerOpts(containers.CreateContainerOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
            Metadata: map[string]string{"owner": "initial"},
        }),
        containers.WithCreateContainerHeaders(map[string]string{"X-Trace-Label": "container-create"}),
        containers.WithCreateContainerHeader("X-Request-Label", "create"),
        containers.WithCreateContainerMetadata(map[string]string{
            "owner": "ops", "note": "literal %2F 雪",
        }),
    }
    created, err := api.CreateContainer(ctx, name, createOptions...)
    if created != nil { fmt.Println(created.StatusCode) }
    if err != nil { return err }

    deleteOptions := []containers.DeleteContainerOption{
        containers.WithDeleteContainerOpts(containers.DeleteContainerOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithDeleteContainerHeaders(map[string]string{"X-Trace-Label": "container-delete"}),
        containers.WithDeleteContainerHeader("X-Request-Label", "delete"),
        containers.WithDeleteContainerIgnoreMissing(false),
    }
    deleted, err := api.DeleteContainer(ctx, name, deleteOptions...)
    if deleted != nil { fmt.Println(deleted.StatusCode, deleted.IgnoredMissing) }
    if err != nil { return err }

    quietOptions := append([]containers.DeleteContainerOption(nil), deleteOptions...)
    quietOptions = append(quietOptions, containers.WithDeleteContainerIgnoreMissing(true))
    missing, err := api.DeleteContainer(ctx, name, quietOptions...)
    if missing != nil { fmt.Println(missing.StatusCode, missing.IgnoredMissing) }
    return err
}

func main() {}
```

## References

- [Listing](../../../docs/listing.md)
- [Objects](../objects/README.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
- [Swift Listing Contracts Test](../../../api/swift_listing_contracts_test.go)
- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L189)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L68)
- [Storage Container Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-container-services.inc#L240)
- [Base](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/base.py#L1953)
- [Server](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/container/server.py#L59)
- [Metadata Contracts Test](metadata_contracts_test.go)
- [Storage Container Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-container-services.inc#L117)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L154)
- [Db](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/db.py#L1015)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1519)
- [Lifecycle Core Test](lifecycle_core_test.go)
- [Lifecycle Options Test](lifecycle_options_test.go)
- [Lifecycle Contracts Test](lifecycle_contracts_test.go)
- [Connection Objectstorage Container Lifecycle Test](../../../connection_objectstorage_container_lifecycle_test.go)
- [Swift Container Lifecycle Test](../../../internal/cmd/sdkgen/swift_container_lifecycle_test.go)
