# Snapshots

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/snapshots.UpdateMetadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.UpdateMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithUpdateMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithUpdateMetadataField`

- PUT snapshots/{id}/metadata accepts native HTTP 200 and returns the actual lowercase metadata object as map[string]any, rather than selecting a promoted Snapshot extractor. A snapshot sibling is ignored; no seeded Snapshot, read/merge/write sequence or follow-up request is added.
  Tests: `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsReturnOnlyMetadataAndExactNativeNumbers`, `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsKeepLiveNativeClientAndSnapshotGet`, `internal/cmd/sdkgen/snapshot_metadata_extractors_test.go:TestSnapshotMetadataExtractorAllowlistChoosesOwnMapWithoutChangingSnapshotCRUD`.
- The already-decoded metadata map retains empty objects, nested objects/arrays, null members and json.Number values without another marshal or decode. This is the native decoded result, not exact raw response bytes or a new status/header model.
  Tests: `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsReturnOnlyMetadataAndExactNativeNumbers`, `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsKeepOmissionAndCoreFieldProtection`, `internal/snapshotmetadata/extract_test.go:TestExtractReturnsCanonicalObjectAndExactNativeNumbers`.
- Absent/null/nonobject canonical metadata or a nonobject root produces a terminal SDK-owned envelope error instead of the pinned native assertion panic. Only exact lowercase metadata is selected, and existing native result.Err takes precedence over response shape.
  Tests: `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsRejectAcceptedWrongEnvelopesWithoutReplay`, `internal/snapshotmetadata/extract_test.go:TestExtractRejectsMalformedEnvelopeWithoutPanicking`, `internal/snapshotmetadata/extract_test.go:TestExtractPreservesOriginalErrorAndConcurrentResponseOwnership`.
- Native unexpected HTTP status, transport, accepted read/decode and context causes remain available through the SDK OperationError. The extractor correction never resends the mutation or automatically fetches the Snapshot after a failure.
  Tests: `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsPreserveNativeFailuresAndPreflight`, `internal/snapshotmetadata/extract_test.go:TestExtractPreservesOriginalErrorAndConcurrentResponseOwnership`.
- The native Metadata map[string]any input and metadata,omitempty builder remain unchanged. Nil and empty maps omit metadata and send an empty root object; populated input is sent with concrete UpdateMetadataOpts, while SDK With functions own builder adaptation and extension-core collision checks.
  Tests: `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsKeepTypedMapOwnershipAndFrozenExtensions`, `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsKeepOmissionAndCoreFieldProtection`.
- Concrete options keep existing sequential replacement and caller-owned typed map behavior. Serialized extension values are snapshotted, follow existing envelope nesting and cannot replace core metadata even when omitted; invalid extension serialization fails before HTTP. Unrelated native operation builders and result types retain their prior policies.
  Tests: `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsKeepTypedMapOwnershipAndFrozenExtensions`, `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsPreserveNativeFailuresAndPreflight`, `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsKeepOmissionAndCoreFieldProtection`, `internal/cmd/sdkgen/snapshot_metadata_extractors_test.go:TestSnapshotMetadataExtractorEmitsSafeMapAndPreservesBuildersAndInventoryPolicy`.
- The request retains the original ServiceClient and Provider, live token, configured headers, selected version and ResourceBase. No endpoint rewrite, project/name discovery, inferred version or extra source policy is added by result extraction.
  Tests: `api/snapshot_metadata_results_test.go:TestSnapshotMetadataResultsKeepLiveNativeClientAndSnapshotGet`.
- Generation activates the specialized result policy on exactly the audited blockstorage/v2 and v3 Snapshot UpdateMetadata operations. Exact native request/options/interface/result/own-vs-promoted method graph, sole Metadata field and complete tag, and six declaration hashes per package fail closed on drift; receiver-qualified source rejects missing/duplicate declarations. Twenty-one other compiled metadata operations retain generic extraction.
  Tests: `internal/cmd/sdkgen/snapshot_metadata_extractors_test.go:TestSnapshotMetadataExtractorAllowlistChoosesOwnMapWithoutChangingSnapshotCRUD`, `internal/cmd/sdkgen/snapshot_metadata_extractors_test.go:TestSnapshotMetadataExtractorRejectsRequestResultAndBuilderGraphDrift`, `internal/cmd/sdkgen/snapshot_metadata_extractors_test.go:TestSnapshotMetadataExtractorGuardsAllSixPinnedDeclarationsAndSourceOwnership`, `internal/cmd/sdkgen/snapshot_metadata_extractors_test.go:TestSnapshotMetadataExtractorRejectsMissingAndDuplicateQualifiedSource`, `internal/cmd/sdkgen/snapshot_metadata_extractors_test.go:TestSnapshotMetadataExtractorEmitsSafeMapAndPreservesBuildersAndInventoryPolicy`, `internal/cmd/sdkgen/snapshot_metadata_extractors_test.go:TestSnapshotMetadataExtractorAcceptsActualPromotedGraphAndKeepsOtherMetadataOperations`.

Python/Go differences:

- The corrected SDK return type is map[string]any; callers expecting *Snapshot must read metadata directly. Other declared Snapshot-returning operations retain their own result types.
- Pinned ExtractMetadata uses unchecked object assertions. The SDK returns an envelope error for invalid accepted shapes while preserving original native errors first; it does not invent raw body, header or status evidence absent from the native Result.
- This review maps only the declared native PUT operation. Pinned Python set_snapshot_metadata performs POST merge and cached Resource update; its explicit metadata envelope, metadata GET/delete family and broader session/cache/lifecycle contracts are separate.
- Typed Metadata maps remain caller-owned references through WithUpdateMetadataOptions; extension values are JSON snapshots. Populated metadata retains the existing envelope nesting for extension fields, and the core metadata key cannot be supplied through WithUpdateMetadataField even when omitted.

### `python:block_storage/v3/delete_snapshot_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.DeleteKeys`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.DeleteMetadataRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataSeed`

- DeleteKeys preserves the Python keys-is-None branch: nil sends one PUT with metadata:{} and returns its actual clear response; a nonnil empty slice validates without HTTP or fabricated body/header/status.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesEmptyMapsAndDeleteBranches`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataMethodsKeepActualEvidenceExplicitEmptyAndInputSnapshots`.
- Nonempty keys preserve order and duplicates as per-key DELETE /snapshots/{id}/metadata/{key}; each HTTP 200 acknowledgement retains the requested key and actual raw Body/Header/StatusCode. The first error stops later keys and returns already acknowledged deletions; missing keys are terminal, no deduplication or transaction is promised.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesEmptyMapsAndDeleteBranches`, `internal/cindermetadata/scope_test.go:TestMetadataDeleteKeepsLiteralKeysOrderDuplicatesAndPartialAcknowledgements`.
- MetadataIn(ctx, resource.ID(id)) constructs a validated fixed ID/collection URL with zero HTTP and retains the original client. ResourceBase changes and response data cannot retarget the scope; each operation and an empty-key call revalidate source and context. The selected source origin/query is validated before optional Name lookup; malformed UTF-8 metadata inputs fail before custom options or HTTP.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
- Independent result ownership and frozen header options support concurrent scope reuse when the source and input maps are not being concurrently written. Existing native parent Get and Snapshot UpdateMetadata APIs remain distinct and unchanged.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `internal/cindermetadata/scope_test.go:TestMetadataConcurrentReuseKeepsRequestAndResponseOwnership`.
- Generator discovery supplements exactly the four audited Volume/Snapshot ID/name collections with MetadataIn; it rejects missing parent bindings and other resources. No new CRUD collection, Backup child route or inferred metadata capability is registered.
  Tests: `internal/cmd/sdkgen/cinder_metadata_scopes_test.go:TestCinderMetadataScopesUseActualCollectionsAndPreserveNativeAPIs`, `internal/cmd/sdkgen/cinder_metadata_scopes_test.go:TestCinderMetadataScopeRejectsMissingParentBindingAndOtherResources`.
- nil keys clear via one PUT metadata:{}; nonnil empty keys perform no HTTP after validating headers/seed. Nonempty keys are snapshotted before options, deleted in original order with duplicate keys retained. No ignore-missing, implicit fetch, or all-key transaction. After each actual DELETE ACK, truthy cached metadata pops that key; absent cached key produces typed Source-shaped post-ACK cache error and actual ResponseError proof, retaining prior completed cache view and stopping subsequent requests. Rejected receipts stay native errors, never accepted receipt entries.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordDeleteNilEmptyOrderedAndCachedPop`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Capture selected volume service before custom MarshalJSON/options, use fixed existing Cinder metadata Scope with literal ID/key routing, reuse declared record descriptor projection and owned seed/header options. JSON snapshots preserve nested null/false/zero and exact numbers; known invalid seed JSON/descriptors fail before HTTP without fake response proof. No Resource GET, name lookup or microversion discovery; selected default version/header presence and absence remain owned across native retries.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSourceCaptureCallbacksAndPreflight`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Source generic raise_from_response accepts actual HTTP100..399 and rejects400..599 with native causes. Accepted partial body/header/status and rejected native receipt preserve Read/Close/context cause and sticky source changes, including change restored during Close. Physical faults prevent replay or cache success. Clean native retries retain the original immutable JSON body and fixed header/version contract; callback refusal/body mutation retains the original rejected status.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordPhysicalFaultsNeverReplayOrSuppress`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordRetrySnapshotAndNativeCauses`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.

Python/Go differences:

- Go uses explicit resource.Ref and a fixed scope returning owned actual HTTP evidence; Python accepts a string ID or existing Resource and returns/mutates that seeded Resource. Explicit Name lookup is a Go convenience and retains the existing native resolver boundary.
- Metadata values are concrete Go strings. Exact lowercase metadata objects and string members are required; Python metadata uses a dict descriptor and its own json/update behavior. Unknown response envelope fields and their original number spelling remain in raw Body, not in a widened typed metadata map.
- Go closes per-call options to SDK-owned headers, preserves original source ownership and exposes context/error explicitly. It does not reproduce injected Python sessions, custom Resource URI/adapter selection or automatic Resource microversion negotiation.
- No new metadata-specific microversion gate is introduced. Fixed Cinder 27.0.0 proves v3 routes and Volume GET/conditional collection PUT ETag at 3.15+, not current legacy v2 deployment support or an equivalent Snapshot ETag policy; caller/server own that condition.
- Literal ID/key components are escaped once and invalid route inputs are rejected. Mock routing evidence is not a guarantee of encoded slash/dot handling through a real proxy/WSGI or of server key/length policy acceptance.
- Python updates the Resource metadata cache after each acknowledged DELETE and can raise from a stale/missing local key; Go records actual partial acknowledgements without a cache mutation. Non-HTTP empty-key results contain no synthesized success evidence, and 404 is not ignored.
- The mapping covers the fixed named literal-ID workflow with optional owned declared-record seed. Existing MetadataIn/Get/Merge/Replace/DeleteKeys string-result APIs and generated/native counts remain unchanged. Python mutable Resource/subclass/dirty-cache/session/Adapter/custom-base-path/connection-location parity remains separately tracked rather than counted by this named mapping.
- GetMetadataRecord is the pinned get_*_metadata alias without emitting Python deprecation warnings. GET requires a UTF-8 JSON object; Python exotic root iterable membership behavior is deliberately outside this bounded metadata endpoint contract. Known declared seed JSON and descriptors are validated before HTTP; Go fixes one escaped literal-ID component, including inner whitespace/slash/percent; blank/dot/control/invalid-UTF8 IDs are rejected before HTTP.
- Go exposes owned actual metadata response receipts separately from the declared cached Resource view. Set caches exactly the requested delta rather than the server merged response. Python delete returns None; Go returns ordered accepted receipts and prior completed cache updates, including an accepted partial response or a post-ACK cache-pop error. ACK/cache state does not establish transaction, durability, replication, policy authorization or subsequent server state.
- Metadata values expose the Source JSON domain rather than enforcing runtime str annotations; server metadata schema/policy still validates arbitrary values. Go JSON-inexpressible values fail locally. Snapshot bounded declared location stays null; Volume reuses existing cloudlocation/project projection and optional owned location without metadata discovery.

### `python:block_storage/v3/fetch_snapshot_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.FetchMetadataRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataSeed`

- Get sends one GET to the fixed /snapshots/{id}/metadata collection and accepts HTTP 200 with the actual exact lowercase metadata string object. No parent GET, implicit metadata fetch, name search for an explicit ID or synthesized parent Resource is added.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataMethodsKeepActualEvidenceExplicitEmptyAndInputSnapshots`.
- The fetched string map, complete raw response Body including unknown envelope fields and precise unknown numbers, independent multi-valued Header and StatusCode belong to the actual response. Caller mutations do not alter later responses or the source client.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `internal/cindermetadata/scope_test.go:TestMetadataMethodsKeepActualEvidenceExplicitEmptyAndInputSnapshots`.
- MetadataIn(ctx, resource.ID(id)) constructs a validated fixed ID/collection URL with zero HTTP and retains the original client. ResourceBase changes and response data cannot retarget the scope; each operation and an empty-key call revalidate source and context. The selected source origin/query is validated before optional Name lookup; malformed UTF-8 metadata inputs fail before custom options or HTTP.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
- Independent result ownership and frozen header options support concurrent scope reuse when the source and input maps are not being concurrently written. Existing native parent Get and Snapshot UpdateMetadata APIs remain distinct and unchanged.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `internal/cindermetadata/scope_test.go:TestMetadataConcurrentReuseKeepsRequestAndResponseOwnership`.
- Generator discovery supplements exactly the four audited Volume/Snapshot ID/name collections with MetadataIn; it rejects missing parent bindings and other resources. No new CRUD collection, Backup child route or inferred metadata capability is registered.
  Tests: `internal/cmd/sdkgen/cinder_metadata_scopes_test.go:TestCinderMetadataScopesUseActualCollectionsAndPreserveNativeAPIs`, `internal/cmd/sdkgen/cinder_metadata_scopes_test.go:TestCinderMetadataScopeRejectsMissingParentBindingAndOtherResources`.
- One bodyless collection-scoped GET; fetch/get aliases return the selected declared Resource record, not a metadata-only dict. Update exactly response metadata member when present, retain seed when omitted, keep explicit null and Source dict descriptor conversion for nonnull scalar/list. Required ID controls record identity; unrelated response id/name/unknown fields remain actual Wire/Envelope evidence only. Record Resource/Wire/Envelope and factory seed bytes do not alias.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordFetchAliasesSeedsAndRawValues`.
- Capture selected volume service before custom MarshalJSON/options, use fixed existing Cinder metadata Scope with literal ID/key routing, reuse declared record descriptor projection and owned seed/header options. JSON snapshots preserve nested null/false/zero and exact numbers; known invalid seed JSON/descriptors fail before HTTP without fake response proof. No Resource GET, name lookup or microversion discovery; selected default version/header presence and absence remain owned across native retries.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSourceCaptureCallbacksAndPreflight`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Source generic raise_from_response accepts actual HTTP100..399 and rejects400..599 with native causes. Accepted partial body/header/status and rejected native receipt preserve Read/Close/context cause and sticky source changes, including change restored during Close. Physical faults prevent replay or cache success. Clean native retries retain the original immutable JSON body and fixed header/version contract; callback refusal/body mutation retains the original rejected status.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordPhysicalFaultsNeverReplayOrSuppress`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordRetrySnapshotAndNativeCauses`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.

Python/Go differences:

- Go uses explicit resource.Ref and a fixed scope returning owned actual HTTP evidence; Python accepts a string ID or existing Resource and returns/mutates that seeded Resource. Explicit Name lookup is a Go convenience and retains the existing native resolver boundary.
- Metadata values are concrete Go strings. Exact lowercase metadata objects and string members are required; Python metadata uses a dict descriptor and its own json/update behavior. Unknown response envelope fields and their original number spelling remain in raw Body, not in a widened typed metadata map.
- Go closes per-call options to SDK-owned headers, preserves original source ownership and exposes context/error explicitly. It does not reproduce injected Python sessions, custom Resource URI/adapter selection or automatic Resource microversion negotiation.
- No new metadata-specific microversion gate is introduced. Fixed Cinder 27.0.0 proves v3 routes and Volume GET/conditional collection PUT ETag at 3.15+, not current legacy v2 deployment support or an equivalent Snapshot ETag policy; caller/server own that condition.
- Literal ID/key components are escaped once and invalid route inputs are rejected. Mock routing evidence is not a guarantee of encoded slash/dot handling through a real proxy/WSGI or of server key/length policy acceptance.
- The mapping covers the fixed named literal-ID workflow with optional owned declared-record seed. Existing MetadataIn/Get/Merge/Replace/DeleteKeys string-result APIs and generated/native counts remain unchanged. Python mutable Resource/subclass/dirty-cache/session/Adapter/custom-base-path/connection-location parity remains separately tracked rather than counted by this named mapping.
- GetMetadataRecord is the pinned get_*_metadata alias without emitting Python deprecation warnings. GET requires a UTF-8 JSON object; Python exotic root iterable membership behavior is deliberately outside this bounded metadata endpoint contract. Known declared seed JSON and descriptors are validated before HTTP; Go fixes one escaped literal-ID component, including inner whitespace/slash/percent; blank/dot/control/invalid-UTF8 IDs are rejected before HTTP.
- Go exposes owned actual metadata response receipts separately from the declared cached Resource view. Set caches exactly the requested delta rather than the server merged response. Python delete returns None; Go returns ordered accepted receipts and prior completed cache updates, including an accepted partial response or a post-ACK cache-pop error. ACK/cache state does not establish transaction, durability, replication, policy authorization or subsequent server state.
- Metadata values expose the Source JSON domain rather than enforcing runtime str annotations; server metadata schema/policy still validates arbitrary values. Go JSON-inexpressible values fail locally. Snapshot bounded declared location stays null; Volume reuses existing cloudlocation/project projection and optional owned location without metadata discovery.

### `python:block_storage/v3/get_snapshot_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.GetMetadataRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataSeed`

- Get sends one GET to the fixed /snapshots/{id}/metadata collection and accepts HTTP 200 with the actual exact lowercase metadata string object. No parent GET, implicit metadata fetch, name search for an explicit ID or synthesized parent Resource is added.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataMethodsKeepActualEvidenceExplicitEmptyAndInputSnapshots`.
- The fetched string map, complete raw response Body including unknown envelope fields and precise unknown numbers, independent multi-valued Header and StatusCode belong to the actual response. Caller mutations do not alter later responses or the source client.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `internal/cindermetadata/scope_test.go:TestMetadataMethodsKeepActualEvidenceExplicitEmptyAndInputSnapshots`.
- MetadataIn(ctx, resource.ID(id)) constructs a validated fixed ID/collection URL with zero HTTP and retains the original client. ResourceBase changes and response data cannot retarget the scope; each operation and an empty-key call revalidate source and context. The selected source origin/query is validated before optional Name lookup; malformed UTF-8 metadata inputs fail before custom options or HTTP.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
- Independent result ownership and frozen header options support concurrent scope reuse when the source and input maps are not being concurrently written. Existing native parent Get and Snapshot UpdateMetadata APIs remain distinct and unchanged.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `internal/cindermetadata/scope_test.go:TestMetadataConcurrentReuseKeepsRequestAndResponseOwnership`.
- Generator discovery supplements exactly the four audited Volume/Snapshot ID/name collections with MetadataIn; it rejects missing parent bindings and other resources. No new CRUD collection, Backup child route or inferred metadata capability is registered.
  Tests: `internal/cmd/sdkgen/cinder_metadata_scopes_test.go:TestCinderMetadataScopesUseActualCollectionsAndPreserveNativeAPIs`, `internal/cmd/sdkgen/cinder_metadata_scopes_test.go:TestCinderMetadataScopeRejectsMissingParentBindingAndOtherResources`.
- One bodyless collection-scoped GET; fetch/get aliases return the selected declared Resource record, not a metadata-only dict. Update exactly response metadata member when present, retain seed when omitted, keep explicit null and Source dict descriptor conversion for nonnull scalar/list. Required ID controls record identity; unrelated response id/name/unknown fields remain actual Wire/Envelope evidence only. Record Resource/Wire/Envelope and factory seed bytes do not alias.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordFetchAliasesSeedsAndRawValues`.
- Capture selected volume service before custom MarshalJSON/options, use fixed existing Cinder metadata Scope with literal ID/key routing, reuse declared record descriptor projection and owned seed/header options. JSON snapshots preserve nested null/false/zero and exact numbers; known invalid seed JSON/descriptors fail before HTTP without fake response proof. No Resource GET, name lookup or microversion discovery; selected default version/header presence and absence remain owned across native retries.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSourceCaptureCallbacksAndPreflight`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Source generic raise_from_response accepts actual HTTP100..399 and rejects400..599 with native causes. Accepted partial body/header/status and rejected native receipt preserve Read/Close/context cause and sticky source changes, including change restored during Close. Physical faults prevent replay or cache success. Clean native retries retain the original immutable JSON body and fixed header/version contract; callback refusal/body mutation retains the original rejected status.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordPhysicalFaultsNeverReplayOrSuppress`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordRetrySnapshotAndNativeCauses`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.

Python/Go differences:

- Go uses explicit resource.Ref and a fixed scope returning owned actual HTTP evidence; Python accepts a string ID or existing Resource and returns/mutates that seeded Resource. Explicit Name lookup is a Go convenience and retains the existing native resolver boundary.
- Metadata values are concrete Go strings. Exact lowercase metadata objects and string members are required; Python metadata uses a dict descriptor and its own json/update behavior. Unknown response envelope fields and their original number spelling remain in raw Body, not in a widened typed metadata map.
- Go closes per-call options to SDK-owned headers, preserves original source ownership and exposes context/error explicitly. It does not reproduce injected Python sessions, custom Resource URI/adapter selection or automatic Resource microversion negotiation.
- No new metadata-specific microversion gate is introduced. Fixed Cinder 27.0.0 proves v3 routes and Volume GET/conditional collection PUT ETag at 3.15+, not current legacy v2 deployment support or an equivalent Snapshot ETag policy; caller/server own that condition.
- Literal ID/key components are escaped once and invalid route inputs are rejected. Mock routing evidence is not a guarantee of encoded slash/dot handling through a real proxy/WSGI or of server key/length policy acceptance.
- The pinned get_snapshot_metadata method is a deprecated alias warning about SDK 5.0 removal and delegating to fetch_metadata. Go reuses MetadataScope.Get and does not add a duplicate deprecated warning-emitting method.
- The mapping covers the fixed named literal-ID workflow with optional owned declared-record seed. Existing MetadataIn/Get/Merge/Replace/DeleteKeys string-result APIs and generated/native counts remain unchanged. Python mutable Resource/subclass/dirty-cache/session/Adapter/custom-base-path/connection-location parity remains separately tracked rather than counted by this named mapping.
- GetMetadataRecord is the pinned get_*_metadata alias without emitting Python deprecation warnings. GET requires a UTF-8 JSON object; Python exotic root iterable membership behavior is deliberately outside this bounded metadata endpoint contract. Known declared seed JSON and descriptors are validated before HTTP; Go fixes one escaped literal-ID component, including inner whitespace/slash/percent; blank/dot/control/invalid-UTF8 IDs are rejected before HTTP.
- Go exposes owned actual metadata response receipts separately from the declared cached Resource view. Set caches exactly the requested delta rather than the server merged response. Python delete returns None; Go returns ordered accepted receipts and prior completed cache updates, including an accepted partial response or a post-ACK cache-pop error. ACK/cache state does not establish transaction, durability, replication, policy authorization or subsequent server state.
- Metadata values expose the Source JSON domain rather than enforcing runtime str annotations; server metadata schema/policy still validates arbitrary values. Go JSON-inexpressible values fail locally. Snapshot bounded declared location stays null; Volume reuses existing cloudlocation/project projection and optional owned location without metadata discovery.

### `python:block_storage/v3/set_snapshot_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.Merge`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.Replace`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.SetMetadataRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithRecordMetadataSeed`

- Merge maps the declared Python set operation to POST /snapshots/{id}/metadata with one explicit metadata object, including nil and empty input as metadata:{}. It returns the full actual server metadata map rather than caching the supplied delta; no read/merge/write or parent GET occurs.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesEmptyMapsAndDeleteBranches`, `internal/cindermetadata/scope_test.go:TestMetadataMethodsKeepActualEvidenceExplicitEmptyAndInputSnapshots`.
- The additional inherited replacement capability is explicit scope.Replace: PUT uses the complete metadata object, with nil/empty maps clearing it. Replace is not another direct proxy declaration, and native Snapshot UpdateMetadata retains its existing omitempty input and decoded map[string]any result.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesEmptyMapsAndDeleteBranches`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `internal/cindermetadata/scope_test.go:TestMetadataMethodsKeepActualEvidenceExplicitEmptyAndInputSnapshots`.
- MetadataIn(ctx, resource.ID(id)) constructs a validated fixed ID/collection URL with zero HTTP and retains the original client. ResourceBase changes and response data cannot retarget the scope; each operation and an empty-key call revalidate source and context. The selected source origin/query is validated before optional Name lookup; malformed UTF-8 metadata inputs fail before custom options or HTTP.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connectiontest/connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
- Independent result ownership and frozen header options support concurrent scope reuse when the source and input maps are not being concurrently written. Existing native parent Get and Snapshot UpdateMetadata APIs remain distinct and unchanged.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `internal/cindermetadata/scope_test.go:TestMetadataConcurrentReuseKeepsRequestAndResponseOwnership`.
- Generator discovery supplements exactly the four audited Volume/Snapshot ID/name collections with MetadataIn; it rejects missing parent bindings and other resources. No new CRUD collection, Backup child route or inferred metadata capability is registered.
  Tests: `internal/cmd/sdkgen/cinder_metadata_scopes_test.go:TestCinderMetadataScopesUseActualCollectionsAndPreserveNativeAPIs`, `internal/cmd/sdkgen/cinder_metadata_scopes_test.go:TestCinderMetadataScopeRejectsMissingParentBindingAndOtherResources`.
- POST metadata merge uses an immutable arbitrary nested JSON map snapshot; nil/empty maps explicitly send metadata:{} before options can mutate caller input. Source cached Resource metadata becomes the exact request delta, not previous cache or response merged state. All successful opaque/empty/null/scalar/list mutation response bodies are preserved without body-driven cache promotion.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSetCachesRequestDeltaAndOwnsInputs`.
- Capture selected volume service before custom MarshalJSON/options, use fixed existing Cinder metadata Scope with literal ID/key routing, reuse declared record descriptor projection and owned seed/header options. JSON snapshots preserve nested null/false/zero and exact numbers; known invalid seed JSON/descriptors fail before HTTP without fake response proof. No Resource GET, name lookup or microversion discovery; selected default version/header presence and absence remain owned across native retries.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSourceCaptureCallbacksAndPreflight`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Source generic raise_from_response accepts actual HTTP100..399 and rejects400..599 with native causes. Accepted partial body/header/status and rejected native receipt preserve Read/Close/context cause and sticky source changes, including change restored during Close. Physical faults prevent replay or cache success. Clean native retries retain the original immutable JSON body and fixed header/version contract; callback refusal/body mutation retains the original rejected status.
  Tests: `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordPhysicalFaultsNeverReplayOrSuppress`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordRetrySnapshotAndNativeCauses`, `blockstorage/v3/snapshots/python_metadata_test.go:TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.

Python/Go differences:

- Go uses explicit resource.Ref and a fixed scope returning owned actual HTTP evidence; Python accepts a string ID or existing Resource and returns/mutates that seeded Resource. Explicit Name lookup is a Go convenience and retains the existing native resolver boundary.
- Metadata values are concrete Go strings. Exact lowercase metadata objects and string members are required; Python metadata uses a dict descriptor and its own json/update behavior. Unknown response envelope fields and their original number spelling remain in raw Body, not in a widened typed metadata map.
- Go closes per-call options to SDK-owned headers, preserves original source ownership and exposes context/error explicitly. It does not reproduce injected Python sessions, custom Resource URI/adapter selection or automatic Resource microversion negotiation.
- No new metadata-specific microversion gate is introduced. Fixed Cinder 27.0.0 proves v3 routes and Volume GET/conditional collection PUT ETag at 3.15+, not current legacy v2 deployment support or an equivalent Snapshot ETag policy; caller/server own that condition.
- Literal ID/key components are escaped once and invalid route inputs are rejected. Mock routing evidence is not a guarantee of encoded slash/dot handling through a real proxy/WSGI or of server key/length policy acceptance.
- Python set_metadata caches the request delta and returns self rather than translating the complete merged response. Go Merge and explicit Replace return server values, have no cached lifecycle, and send metadata:{} for nil/empty inputs; native Snapshot omitempty behavior remains separate.
- The mapping covers the fixed named literal-ID workflow with optional owned declared-record seed. Existing MetadataIn/Get/Merge/Replace/DeleteKeys string-result APIs and generated/native counts remain unchanged. Python mutable Resource/subclass/dirty-cache/session/Adapter/custom-base-path/connection-location parity remains separately tracked rather than counted by this named mapping.
- GetMetadataRecord is the pinned get_*_metadata alias without emitting Python deprecation warnings. GET requires a UTF-8 JSON object; Python exotic root iterable membership behavior is deliberately outside this bounded metadata endpoint contract. Known declared seed JSON and descriptors are validated before HTTP; Go fixes one escaped literal-ID component, including inner whitespace/slash/percent; blank/dot/control/invalid-UTF8 IDs are rejected before HTTP.
- Go exposes owned actual metadata response receipts separately from the declared cached Resource view. Set caches exactly the requested delta rather than the server merged response. Python delete returns None; Go returns ordered accepted receipts and prior completed cache updates, including an accepted partial response or a post-ACK cache-pop error. ACK/cache state does not establish transaction, durability, replication, policy authorization or subsequent server state.
- Metadata values expose the Source JSON domain rather than enforcing runtime str annotations; server metadata schema/policy still validates arbitrary values. Go JSON-inexpressible values fail locally. Snapshot bounded declared location stays null; Volume reuses existing cloudlocation/project projection and optional owned location without metadata discovery.

### `python:block_storage/v3/wait_for_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForAvailable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.WaitForAvailable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.WaitForAvailable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/capabilities.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.QuotaClassResources`

- Cinder WaitForState requires a target and WaitForAvailable supplies available; both apply exact case-insensitive error, two-second polling and no SDK timeout. The returned model is the actual native observation, not a cached supplied Python Resource.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`.
- Target state comparison precedes failure comparison. WithFailureStates replaces the exact case-insensitive default list, and an empty list disables state failure checks. error_deleting/error_extending, failed or killed are not inferred as additional service failures. Captured failure-state values remain independent of later caller slice mutations.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitExactFailuresReplacementAndTargetPrecedence`, `api/service_wait_contracts_test.go:TestServiceWaitContractsExactFailuresAndCallerOverrides`.
- SDK defaults are prepended and concrete caller resource.WaitOption values apply last. WithUnlimitedWait removes a finite default, a later positive WithTimeout reinstates a bound, and parent context deadlines and cancellation remain effective. No application status adapter or polling builder is required.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`.
- The default polling interval is two seconds and a positive caller interval replaces it. Nil context or collection, invalid refs/targets/options, unsupported selected attributes and canceled context fail without HTTP. Callback cancellation and active-fetch cancellation stop before any further poll.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDefaultTwoSecondIntervalAndCallerShortening`, `internal/servicewait/wait_test.go:TestServiceWaitPreflightAndCallerCancellationBeforeAnotherFetch`, `api/service_wait_contracts_test.go:TestServiceWaitContractsPreflightAndCallerCancellation`.
- Status selection uses an exported string field and can also select deletion completion. Server canonical integer progress reports native values; absent canonical progress reports zero. Cinder-v3 Snapshot uses the noncanonical os-extended-snapshot-attributes:progress string tag, so callback reports zero without parsing that string. Successful terminal, failure and HTTP-error observations do not invoke the callback.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitReusableCallerOptionsRemainOwnedAndSelectAttributes`, `api/service_wait_contracts_test.go:TestServiceWaitContractsStatusAttributesAndNativeProgressLimits`, `api/service_wait_contracts_test.go:TestServiceWaitContractsNameBindingLiveClientAndProgress`, `resource/wait_attributes_test.go:TestWaitProgressCallbackCancellationStopsFurtherGETs`, `resource/wait_attributes_test.go:TestWaitProgressOmitsSuccessFailureAndHTTPError`.
- Explicit ID is stable even when response ID differs. Existing Name lookup resolves once, retains the native list/name-query boundary, and state waiting reports its initial List observation before fixed-ID GETs. Selected ResourceBase/reverse prefix, configured ordinary headers/microversion, caller context/middleware and the original live Provider token remain in use; no response-driven route or model synthesis occurs.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsNameBindingLiveClientAndProgress`, `resource/wait_identity_test.go:TestWaitKeepsExplicitIDWhenResponseIdentityChanges`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`.
- Fresh lookup failures remain errors rather than a successful target model. The shared read gate preserves original terminal URL/accepted-response/JSON/read/context causes even when nested native404 or adapter-owned ErrNotFound is discoverable. Actual direct403 or transport-wrapped404 terminates the service deletion observation after one request. Successful missing state fetches return a nonpanic error; successful nil deletion fetches complete.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`, `resource/read_errors_test.go:TestReadErrorsNeverAddMissingToTerminalDirectOrQueryCauses`, `resource/read_errors_test.go:TestReadErrorsDefendIgnoreMissingAndWaitAgainstAdapterPrewrappedTerminalCauses`, `resource/wait_policy_test.go:TestWaitMissingSuccessfulResponseDoesNotPanic`, `resource/wait_attributes_test.go:TestWaitDeletedNilSuccessfulFetchIsTerminalWithoutCallback`.
- Existing common Collection.Wait/WaitDeleted and generated WaitFor/WaitForDeletion retain their five-minute deadline and original binding failure predicate. Native WaitForStatus retains its error-only callable and native predicate-first behavior, including original caller context; source one-second/exact-case behavior is not replaced by this service policy. This is preservation evidence, not a separate native-operation status promotion.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsLegacyAndNativeSignaturesStayUnchanged`, `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`.
- Generator discovery records only the six audited Server/Cinder Volume-Snapshot/Image-v2 collections with their service wait policy. Existing generated common WaitFor/WaitForDeletion still delegate unchanged; unrelated Backup, Flavor, Task, network and Image-v1 collections receive no inferred service-default policy.
  Tests: `internal/cmd/sdkgen/service_waits_test.go:TestServiceWaitPoliciesBindOnlyActualStatusCollections`, `internal/cmd/sdkgen/service_waits_test.go:TestServiceWaitPolicyRejectsParentDriftAndDoesNotInferOtherModels`.
- Group, group snapshot, consistency group and consistency group snapshot records and clusters expose their status to blockstorage.WaitForState (case-insensitive target, error failure, deleted completion); status-less models fail before HTTP without a selected attribute.
  Tests: `api/cinder_proxy_waits_test.go:TestCinderProxyWaitsRecordAndClusterStatus`, `api/cinder_proxy_waits_test.go:TestCinderProxyWaitsDeleteEveryFetchableResource`.
- Records that keep their declared Resource view as raw JSON (group, group snapshot, consistency group, consistency group snapshot, message, capabilities and quota class records) accept resource.WithStatusAttribute with an exact Python attribute name. The view value is compared case-insensitively with the same default and caller failure states as typed fields, and a selected attribute also drives the deleted check of deletion waits. A wire spelling that differs from the attribute name, an undeclared member of a class that discards unknown Body attributes, and an absent, null, numeric, boolean or object value fail with resource.ErrUnsupported. Group-family records keep their default status accessor when no attribute is selected.
  Tests: `api/raw_record_wait_attributes_test.go:TestRawRecordWaitAttributesCinder`, `resource/wait_raw_attributes_test.go:TestWaitRawAttributeComparesViewStringsLikeTypedFields`, `resource/wait_raw_attributes_test.go:TestWaitRawAttributeRejectsMissingNullAndNonStringValues`, `resource/wait_raw_attributes_test.go:TestWaitRawAttributeKeepsBindingStatusDefaultAndDeletion`.

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.
- Python nullable descriptor/status/progress coercion and Resource-return behavior are outside the explicit Ref/native-model API (see the differences above); native name-resolution paging limits and model decoders remain unchanged.
- Image-v1, Glance wait_for_task and Cloud wait_for_server/wait_for_image workflows are separate declarations and are not promoted by this operation.
- Limits and BlockStorageSummary have no member ID; Python's inherited fetch requires one, so neither is waitable in Python or Go.
- For records that keep a raw JSON Resource view, a selected attribute is read from each observation: an attribute the view does not carry, a null value or a non-string value fails with resource.ErrUnsupported after the first GET rather than before HTTP, and a null value fails instead of being polled like Python's None until the timeout. Only group-family records select status by default; message, capabilities and quota class records need resource.WithStatusAttribute. Progress callbacks for these records report zero.

### `python:block_storage/v3/wait_for_delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/capabilities.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.QuotaClassResources`

- SDK-owned WaitForDelete supplies a two-second interval and120-second SDK timeout. Actual404, successful nil or deleted status complete with error-only success. An ERROR observation does not itself complete or fail this deletion waiter; caller failure-state lists do not change deletion-state policy.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`, `internal/servicewait/wait_test.go:TestServiceWaitDeleteObservesAbsenceWithoutFailureOrMutation`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`, `resource/wait_attributes_test.go:TestWaitDeletedNilSuccessfulFetchIsTerminalWithoutCallback`.
- Deletion waiting sends GET observations and optional initial Name resolution only, never a DELETE request. Name is resolved once without reporting its resolution List to the callback, then the first fixed-ID GET and later nonterminal GETs can report progress. Changed response IDs cannot retarget the poll.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`, `internal/servicewait/wait_test.go:TestServiceWaitDeleteObservesAbsenceWithoutFailureOrMutation`, `resource/wait_identity_test.go:TestWaitKeepsExplicitIDWhenResponseIdentityChanges`.
- SDK defaults are prepended and concrete caller resource.WaitOption values apply last. WithUnlimitedWait removes a finite default, a later positive WithTimeout reinstates a bound, and parent context deadlines and cancellation remain effective. No application status adapter or polling builder is required.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`.
- The default polling interval is two seconds and a positive caller interval replaces it. Nil context or collection, invalid refs/targets/options, unsupported selected attributes and canceled context fail without HTTP. Callback cancellation and active-fetch cancellation stop before any further poll.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDefaultTwoSecondIntervalAndCallerShortening`, `internal/servicewait/wait_test.go:TestServiceWaitPreflightAndCallerCancellationBeforeAnotherFetch`, `api/service_wait_contracts_test.go:TestServiceWaitContractsPreflightAndCallerCancellation`.
- Status selection uses an exported string field and can also select deletion completion. Server canonical integer progress reports native values; absent canonical progress reports zero. Cinder-v3 Snapshot uses the noncanonical os-extended-snapshot-attributes:progress string tag, so callback reports zero without parsing that string. Successful terminal, failure and HTTP-error observations do not invoke the callback.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitReusableCallerOptionsRemainOwnedAndSelectAttributes`, `api/service_wait_contracts_test.go:TestServiceWaitContractsStatusAttributesAndNativeProgressLimits`, `api/service_wait_contracts_test.go:TestServiceWaitContractsNameBindingLiveClientAndProgress`, `resource/wait_attributes_test.go:TestWaitProgressCallbackCancellationStopsFurtherGETs`, `resource/wait_attributes_test.go:TestWaitProgressOmitsSuccessFailureAndHTTPError`.
- Explicit ID is stable even when response ID differs. Existing Name lookup resolves once, retains the native list/name-query boundary, and state waiting reports its initial List observation before fixed-ID GETs. Selected ResourceBase/reverse prefix, configured ordinary headers/microversion, caller context/middleware and the original live Provider token remain in use; no response-driven route or model synthesis occurs.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsNameBindingLiveClientAndProgress`, `resource/wait_identity_test.go:TestWaitKeepsExplicitIDWhenResponseIdentityChanges`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`.
- Terminal failures remain failures rather than deletion completion. The shared read gate preserves original terminal URL/accepted-response/JSON/read/context causes even when nested native404 or adapter-owned ErrNotFound is discoverable. Actual direct403 or transport-wrapped404 terminates the service deletion observation after one request. Successful missing state fetches return a nonpanic error; successful nil deletion fetches complete.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`, `resource/read_errors_test.go:TestReadErrorsNeverAddMissingToTerminalDirectOrQueryCauses`, `resource/read_errors_test.go:TestReadErrorsDefendIgnoreMissingAndWaitAgainstAdapterPrewrappedTerminalCauses`, `resource/wait_policy_test.go:TestWaitMissingSuccessfulResponseDoesNotPanic`, `resource/wait_attributes_test.go:TestWaitDeletedNilSuccessfulFetchIsTerminalWithoutCallback`.
- Existing common Collection.Wait/WaitDeleted and generated WaitFor/WaitForDeletion retain their five-minute deadline and original binding failure predicate. Native WaitForStatus retains its error-only callable and native predicate-first behavior, including original caller context; source one-second/exact-case behavior is not replaced by this service policy. This is preservation evidence, not a separate native-operation status promotion.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsLegacyAndNativeSignaturesStayUnchanged`, `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`.
- Generator discovery records only the six audited Server/Cinder Volume-Snapshot/Image-v2 collections with their service wait policy. Existing generated common WaitFor/WaitForDeletion still delegate unchanged; unrelated Backup, Flavor, Task, network and Image-v1 collections receive no inferred service-default policy.
  Tests: `internal/cmd/sdkgen/service_waits_test.go:TestServiceWaitPoliciesBindOnlyActualStatusCollections`, `internal/cmd/sdkgen/service_waits_test.go:TestServiceWaitPolicyRejectsParentDriftAndDoesNotInferOtherModels`.
- Every pinned fetchable block_storage/v3 Resource with a member route (volumes, snapshots, backups, attachments, transfers, QoS specs, types, groups, group snapshots, consistency groups and snapshots, clusters, default types, group types, messages, capabilities and quota class sets) has an SDK collection; blockstorage.WaitForDelete polls each fixed member route until 404, and record collections complete on a deleted status.
  Tests: `api/cinder_proxy_waits_test.go:TestCinderProxyWaitsDeleteEveryFetchableResource`, `api/cinder_proxy_waits_test.go:TestCinderProxyWaitsRecordAndClusterStatus`.

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.
- Python nullable descriptor/status/progress coercion and Resource-return behavior are outside the explicit Ref/native-model API (see the differences above); native name-resolution paging limits and model decoders remain unchanged.
- Image-v1, Glance wait_for_task and Cloud wait_for_server/wait_for_image workflows are separate declarations and are not promoted by this operation.
- Limits and BlockStorageSummary have no member ID; Python's inherited fetch requires one, so neither is waitable in Python or Go.

### `python:block_storage/v3/manage_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.ManageRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithManageRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithManageRecordMetadata`

- Manage checks the advertised and selected-default 3.8 capability to choose the modern or legacy route, then posts with the Source fixed 3.65 version. All five snapshot envelope fields are present including optional nulls; discovery and snapshot physical responses are separate.
  Tests: `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManageRouteCapabilityAndFixedVersion`, `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManagementRetriesOwnFixedVersionAndHeaders`.
- Raw reference and metadata plus optional text are cloned before callbacks; response fields are projected from the actual response without echoing submitted attributes, and missing response id remains Source null. Invalid JSON fails before HTTP; accepted descriptor errors retain the actual response and never expose a successful Resource.
  Tests: `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManageResponseMissingIDAndOptionsOwned`, `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManagementGuardsAndAcceptedProjectionError`.

Python/Go differences:

- Go accepts a string volume ID and finite raw JSON reference/metadata with concrete optional text/header options instead of arbitrary Python kwargs or Resource objects. Unknown formal kwargs cannot be injected.
- The SDK reuses bounded guarded Cinder discovery for supports_microversion; actual discovery observations accompany results rather than Python adapter endpoint-data caching. The Source mismatch between legacy route selection and fixed 3.65 submission is intentionally preserved.
- The result is an owned SnapshotRecord, not a mutable Python Snapshot. Generic dirty lifecycle/cache and automatic Connection location remain global SDK work; no live cloud or storage backend completion is claimed.

### `python:block_storage/v3/unmanage_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.UnmanageRecord`

- Unmanage POSTs the literal os-unmanage:null action to one escaped snapshot member with the Source fixed 3.65 header, ignores response JSON and returns actual opaque acknowledgement. A physical 404 remains a native error, never ignored missing or a successful state transition.
  Tests: `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotUnmanageOpaqueAckFixedHeaderAndMissing`, `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManagementRetriesOwnFixedVersionAndHeaders`.
- Nil options, invalid identifiers and cancelled context stop before HTTP; selected API/source and physical response guards remain terminal rather than falling back or inventing a successful resource.
  Tests: `blockstorage/v3/snapshots/python_manage_test.go:TestPythonSnapshotManagementGuardsAndAcceptedProjectionError`.

Python/Go differences:

- Python returns None; Go returns an owned ManagementReceipt for actual opaque body/header/status. Submission acknowledgement does not prove a completed backend transition.
- Go uses a validated literal string ID escaped as one path segment and owned header options; arbitrary Python Resource/URI/session objects are not accepted. Source fixed 3.65 action behavior is preserved without version discovery.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
)

func replaceSnapshotMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
    return snapshots.New(client).UpdateMetadata(ctx, id, snapshots.UpdateMetadataOpts{
        Metadata: map[string]any{"owner": "team-a", "purpose": "backup"},
    })
}
```

## References

- [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py)
- [snapshot proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [v2 example](../../v2/snapshots/README.md)
- [Generator documentation](../../../internal/cmd/sdkgen/README.md)
- [HTTP contract tests](../../../api/snapshot_metadata_results_test.go)
- [shared guide](../../metadata/README.md)
- [Scoped HTTP tests](../../../api/cinder_metadata_scopes_test.go)
- [Connection tests](../../../connectiontest/connection_metadata_test.go)
- [release-pinned server contract](../../../docs/cinder-metadata-server-contracts.md)
- [Service Waits](../../../docs/service-waits.md)
- [Python Manage](python-manage.md)
