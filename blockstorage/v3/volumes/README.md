# Volumes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/delete_volume_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.DeleteKeys`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.DeleteMetadataRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataSeed`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataLocation`

- DeleteKeys preserves the Python keys-is-None branch: nil sends one PUT with metadata:{} and returns its actual clear response; a nonnil empty slice validates without HTTP or fabricated body/header/status.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesEmptyMapsAndDeleteBranches`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataMethodsKeepActualEvidenceExplicitEmptyAndInputSnapshots`.
- Nonempty keys preserve order and duplicates as per-key DELETE /volumes/{id}/metadata/{key}; each HTTP 200 acknowledgement retains the requested key and actual raw Body/Header/StatusCode. The first error stops later keys and returns already acknowledged deletions; missing keys are terminal, no deduplication or transaction is promised.
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
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordDeleteNilEmptyOrderedAndCachedPop`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Capture selected volume service before custom MarshalJSON/options, use fixed existing Cinder metadata Scope with literal ID/key routing, reuse declared record descriptor projection and owned seed/header options. JSON snapshots preserve nested null/false/zero and exact numbers; known invalid seed JSON/descriptors fail before HTTP without fake response proof. No Resource GET, name lookup or microversion discovery; selected default version/header presence and absence remain owned across native retries.
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSourceCaptureCallbacksAndPreflight`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Source generic raise_from_response accepts actual HTTP100..399 and rejects400..599 with native causes. Accepted partial body/header/status and rejected native receipt preserve Read/Close/context cause and sticky source changes, including change restored during Close. Physical faults prevent replay or cache success. Clean native retries retain the original immutable JSON body and fixed header/version contract; callback refusal/body mutation retains the original rejected status.
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordPhysicalFaultsNeverReplayOrSuppress`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordRetrySnapshotAndNativeCauses`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.

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

### `python:block_storage/v3/fetch_volume_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.FetchMetadataRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataSeed`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataLocation`

- Get sends one GET to the fixed /volumes/{id}/metadata collection and accepts HTTP 200 with the actual exact lowercase metadata string object. No parent GET, implicit metadata fetch, name search for an explicit ID or synthesized parent Resource is added.
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
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordFetchAliasesSeedsAndRawValues`.
- Capture selected volume service before custom MarshalJSON/options, use fixed existing Cinder metadata Scope with literal ID/key routing, reuse declared record descriptor projection and owned seed/header options. JSON snapshots preserve nested null/false/zero and exact numbers; known invalid seed JSON/descriptors fail before HTTP without fake response proof. No Resource GET, name lookup or microversion discovery; selected default version/header presence and absence remain owned across native retries.
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSourceCaptureCallbacksAndPreflight`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Source generic raise_from_response accepts actual HTTP100..399 and rejects400..599 with native causes. Accepted partial body/header/status and rejected native receipt preserve Read/Close/context cause and sticky source changes, including change restored during Close. Physical faults prevent replay or cache success. Clean native retries retain the original immutable JSON body and fixed header/version contract; callback refusal/body mutation retains the original rejected status.
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordPhysicalFaultsNeverReplayOrSuppress`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordRetrySnapshotAndNativeCauses`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.

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

### `python:block_storage/v3/get_volume_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.GetMetadataRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataSeed`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataLocation`

- Get sends one GET to the fixed /volumes/{id}/metadata collection and accepts HTTP 200 with the actual exact lowercase metadata string object. No parent GET, implicit metadata fetch, name search for an explicit ID or synthesized parent Resource is added.
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
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordFetchAliasesSeedsAndRawValues`.
- Capture selected volume service before custom MarshalJSON/options, use fixed existing Cinder metadata Scope with literal ID/key routing, reuse declared record descriptor projection and owned seed/header options. JSON snapshots preserve nested null/false/zero and exact numbers; known invalid seed JSON/descriptors fail before HTTP without fake response proof. No Resource GET, name lookup or microversion discovery; selected default version/header presence and absence remain owned across native retries.
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSourceCaptureCallbacksAndPreflight`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Source generic raise_from_response accepts actual HTTP100..399 and rejects400..599 with native causes. Accepted partial body/header/status and rejected native receipt preserve Read/Close/context cause and sticky source changes, including change restored during Close. Physical faults prevent replay or cache success. Clean native retries retain the original immutable JSON body and fixed header/version contract; callback refusal/body mutation retains the original rejected status.
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordPhysicalFaultsNeverReplayOrSuppress`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordRetrySnapshotAndNativeCauses`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.

Python/Go differences:

- Go uses explicit resource.Ref and a fixed scope returning owned actual HTTP evidence; Python accepts a string ID or existing Resource and returns/mutates that seeded Resource. Explicit Name lookup is a Go convenience and retains the existing native resolver boundary.
- Metadata values are concrete Go strings. Exact lowercase metadata objects and string members are required; Python metadata uses a dict descriptor and its own json/update behavior. Unknown response envelope fields and their original number spelling remain in raw Body, not in a widened typed metadata map.
- Go closes per-call options to SDK-owned headers, preserves original source ownership and exposes context/error explicitly. It does not reproduce injected Python sessions, custom Resource URI/adapter selection or automatic Resource microversion negotiation.
- No new metadata-specific microversion gate is introduced. Fixed Cinder 27.0.0 proves v3 routes and Volume GET/conditional collection PUT ETag at 3.15+, not current legacy v2 deployment support or an equivalent Snapshot ETag policy; caller/server own that condition.
- Literal ID/key components are escaped once and invalid route inputs are rejected. Mock routing evidence is not a guarantee of encoded slash/dot handling through a real proxy/WSGI or of server key/length policy acceptance.
- The pinned get_volume_metadata method is a deprecated alias warning about SDK 5.0 removal and delegating to fetch_metadata. Go reuses MetadataScope.Get and does not add a duplicate deprecated warning-emitting method.
- The mapping covers the fixed named literal-ID workflow with optional owned declared-record seed. Existing MetadataIn/Get/Merge/Replace/DeleteKeys string-result APIs and generated/native counts remain unchanged. Python mutable Resource/subclass/dirty-cache/session/Adapter/custom-base-path/connection-location parity remains separately tracked rather than counted by this named mapping.
- GetMetadataRecord is the pinned get_*_metadata alias without emitting Python deprecation warnings. GET requires a UTF-8 JSON object; Python exotic root iterable membership behavior is deliberately outside this bounded metadata endpoint contract. Known declared seed JSON and descriptors are validated before HTTP; Go fixes one escaped literal-ID component, including inner whitespace/slash/percent; blank/dot/control/invalid-UTF8 IDs are rejected before HTTP.
- Go exposes owned actual metadata response receipts separately from the declared cached Resource view. Set caches exactly the requested delta rather than the server merged response. Python delete returns None; Go returns ordered accepted receipts and prior completed cache updates, including an accepted partial response or a post-ACK cache-pop error. ACK/cache state does not establish transaction, durability, replication, policy authorization or subsequent server state.
- Metadata values expose the Source JSON domain rather than enforcing runtime str annotations; server metadata schema/policy still validates arbitrary values. Go JSON-inexpressible values fail locally. Snapshot bounded declared location stays null; Volume reuses existing cloudlocation/project projection and optional owned location without metadata discovery.

### `python:block_storage/v3/set_volume_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.Merge`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.Replace`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SetMetadataRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataSeed`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithRecordMetadataLocation`

- Merge maps the declared Python set operation to POST /volumes/{id}/metadata with one explicit metadata object, including nil and empty input as metadata:{}. It returns the full actual server metadata map rather than caching the supplied delta; no read/merge/write or parent GET occurs.
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
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSetCachesRequestDeltaAndOwnsInputs`.
- Capture selected volume service before custom MarshalJSON/options, use fixed existing Cinder metadata Scope with literal ID/key routing, reuse declared record descriptor projection and owned seed/header options. JSON snapshots preserve nested null/false/zero and exact numbers; known invalid seed JSON/descriptors fail before HTTP without fake response proof. No Resource GET, name lookup or microversion discovery; selected default version/header presence and absence remain owned across native retries.
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSourceCaptureCallbacksAndPreflight`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.
- Source generic raise_from_response accepts actual HTTP100..399 and rejects400..599 with native causes. Accepted partial body/header/status and rejected native receipt preserve Read/Close/context cause and sticky source changes, including change restored during Close. Physical faults prevent replay or cache success. Clean native retries retain the original immutable JSON body and fixed header/version contract; callback refusal/body mutation retains the original rejected status.
  Tests: `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordHTTPPoliciesAndPartialDeletion`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordPhysicalFaultsNeverReplayOrSuppress`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordRetrySnapshotAndNativeCauses`, `blockstorage/v3/volumes/python_metadata_test.go:TestPythonVolumeMetadataRecordSelectedMicroversionRetryAndSeedPreflight`.

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

### `python:block_storage/v3/reserve_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ReserveVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ReserveVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ReserveVolume`

- ReserveVolume sends exactly {"os-reserve":null}. Use one explicit safe nonempty UTF-8 volume ID and the exact pinned null action through queryless POST volumes/{ID}/action. Unicode literal IDs are preserved, and VolumeDetail never inserts detail into the route. There is no UUID-only restriction, lookup, Resource conversion, status enum, options/defaults, builder, wait, refresh or rollback.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsPostSourceNullBodiesAndAcceptOpaqueHTTPReplies`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsRejectInvalidIDAndSourceBeforeAnyHTTP`.
- Direct calls capture the Cinder service and validate context/role/provider/endpoint/resource/header before ID admission. Connection checks context/provider/ID before selecting only cached Cinder v3, without CurrentLocation or recorded scope consumption. Getter errors retain the actual outer operation and custom cancellation causes before any request or fabricated proof.
  Tests: `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsUseOnlyCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsPreflightNeverSelectsCinder`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsGetterErrorsKeepOperationAndCancellation`.
- Selected nonempty microversion wins literally without discovery/cap. Otherwise operation-owned finite negotiation uses the Volume3.71 ceiling and server maximum before optional minimum, with tuple length/order retained; minimum above ceiling omits version and missing maximum ignores unused minimum. Original client/cache stays unchanged, each call owns its selected policy, and independent Backup import retains3.64 on the same server advertisement.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsSelectedVersionKeepsOwnedHeadersAndLiveNativeAuthentication`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsOwnDiscoveryPerCallAndKeepStageProof`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentStringAndHeaderCases`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentMaximumMinimumCases`.
- Finite discovery preserves captured encoded reverse-proxy prefix, admits200/300 supported body forms and first usable v3, and keeps admitted pages separate from action Applied. Only clean concrete native404/405 permits unavailable fallback. Accepted malformed/null/invalid-UTF8 or other discovery failures stop before action, with current ResponseError proof; later native rejection cannot borrow an earlier admitted page as current error proof.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsDiscoveryFailuresRetainCurrentStageAndNeverBorrowProof`.
- Action replies are opaque finite HTTP100..399 acknowledgements including empty, malformed and invalid-UTF8 bytes. Applied owns actual body/header/status, independently of separate calls or error proof. Completed means accepted Read/Close/source validation succeeded, never a server status postcondition. Original-policy>=400 and expanded native OkCodes retain native rejected evidence without Applied or Completed. Existing native Reserve/Unreserve/BeginDetaching keep their {} body and narrower status policy; all four new service methods and nil API preserve actual labels.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionAdmissionUsesOriginalSub400PolicyAndKeepsNativeRejections`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionServiceMethodsPreserveNativeBodiesCodesAndNilAPILabels`.
- Accepted Read/Close, captured-source drift and custom cancellation retain actual Applied and every joined cause, with Completed=false and an independently owned ResponseError. Body closes once and accepted IO errors invoke no caller retry or SDK replay; proof storage never aliases the caller response or another result.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsAcceptedFaultsKeepActualProofAndAllCauses`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- Live native401 reauthentication and bounded503 caller retry retain exact owned null body, captured ordinary headers, selected source and operation version. Body/decoder/retention changes, version conflicts or source drift stop before resend while original/callback/custom cancellation causes remain reachable.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionLiveReauthAndBoundedRetryKeepLiteralNullBody`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionRetryMutationStopsBodyVersionAndSourceBeforeResend`.
- Every physical POST including same-target307 redirects verifies exact method/path/origin, body bytes/framing and modern/legacy version policy. Changed body/framing/header/target stops before second transport attempt. Physical body Read/Close and header faults remain sticky after caller repair, close inspected body once and preserve actual causes.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionPhysicalRedirectMutationRemainsStickyAfterRepair`.

Python/Go differences:

- Go takes safe literal UTF-8 ID only. Source accepts mutable Volume/VolumeDetail/Resource/dict/Munch inputs through _get_resource with constructor/to_dict/descriptor/computed location side effects and Python urljoin coercion. Go skips those identities/side effects and name lookup deliberately; this does not classify separate Resource declarations.
- Source Proxy and Resource methods return None without translating response or updating local fields. Go adds independently owned Microversion, Discovery, Applied and Completed acknowledgement with literal input VolumeID. No semantic server-state confirmation, wait, follow-up GET or rollback is added.
- Volume uses class ceiling3.71 while Backup import keeps3.64 through the shared Cinder policy. Selected nonempty version is forwarded literally; Keystoneauth Session canonicalization, dynamic nonstring defaults, full Adapter cache/link/envelope/header discovery and deployment HTTP behavior are outside this owned Go mapping.
- Finite same-origin body-only first usable v3 discovery, typed advertisement shapes/status and clean concrete404/405 fallback are explicit mapping policy. Accepted malformed discovery is terminal with current proof rather than reproducing every caught Keystoneauth DiscoveryFailure/None path or endpoint-data exception order.
- Utility tuple selection retains maximum-before-optional-minimum, length/order and missing bounds. Shared arbitrary-precision Unicode16 decimal/underscore string parsing and latest handling are bounded Go behavior; exact installed CPython digit limits, nonstring input normalization and pinned dependency runtime are not claimed.
- Owned finite HTTP100..399 policy preserves opaque action bytes and admitted IO/source/context proof separately from native>=400 rejection. Native discarded rejected POST body Read/Close faults remain a documented limitation; mutable OkCodes cannot widen SDK acknowledgement. No response model conversion is introduced.
- Fixed source/method/route/body/framing and operation microversion ownership constrain mutable native retry/redirect hooks while live authentication and bounded caller retry remain. Native Reserve/Unreserve/BeginDetaching keep {} and narrower success codes; new null action APIs remain separate.
- Only the actual pinned v3 Proxy declaration is mapped here. Native/v2/Resource/inherited targets and direct bootable/readonly setters remain independent. Source hashes, static review, Go compile and offline HTTP/race tests do not establish authenticated OpenStack or Python SDK runtime parity or full SDK completion.

### `python:block_storage/v3/unreserve_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.UnreserveVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UnreserveVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.UnreserveVolume`

- UnreserveVolume sends exactly {"os-unreserve":null}. Use one explicit safe nonempty UTF-8 volume ID and the exact pinned null action through queryless POST volumes/{ID}/action. Unicode literal IDs are preserved, and VolumeDetail never inserts detail into the route. There is no UUID-only restriction, lookup, Resource conversion, status enum, options/defaults, builder, wait, refresh or rollback.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsPostSourceNullBodiesAndAcceptOpaqueHTTPReplies`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsRejectInvalidIDAndSourceBeforeAnyHTTP`.
- Direct calls capture the Cinder service and validate context/role/provider/endpoint/resource/header before ID admission. Connection checks context/provider/ID before selecting only cached Cinder v3, without CurrentLocation or recorded scope consumption. Getter errors retain the actual outer operation and custom cancellation causes before any request or fabricated proof.
  Tests: `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsUseOnlyCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsPreflightNeverSelectsCinder`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsGetterErrorsKeepOperationAndCancellation`.
- Selected nonempty microversion wins literally without discovery/cap. Otherwise operation-owned finite negotiation uses the Volume3.71 ceiling and server maximum before optional minimum, with tuple length/order retained; minimum above ceiling omits version and missing maximum ignores unused minimum. Original client/cache stays unchanged, each call owns its selected policy, and independent Backup import retains3.64 on the same server advertisement.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsSelectedVersionKeepsOwnedHeadersAndLiveNativeAuthentication`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsOwnDiscoveryPerCallAndKeepStageProof`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentStringAndHeaderCases`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentMaximumMinimumCases`.
- Finite discovery preserves captured encoded reverse-proxy prefix, admits200/300 supported body forms and first usable v3, and keeps admitted pages separate from action Applied. Only clean concrete native404/405 permits unavailable fallback. Accepted malformed/null/invalid-UTF8 or other discovery failures stop before action, with current ResponseError proof; later native rejection cannot borrow an earlier admitted page as current error proof.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsDiscoveryFailuresRetainCurrentStageAndNeverBorrowProof`.
- Action replies are opaque finite HTTP100..399 acknowledgements including empty, malformed and invalid-UTF8 bytes. Applied owns actual body/header/status, independently of separate calls or error proof. Completed means accepted Read/Close/source validation succeeded, never a server status postcondition. Original-policy>=400 and expanded native OkCodes retain native rejected evidence without Applied or Completed. Existing native Reserve/Unreserve/BeginDetaching keep their {} body and narrower status policy; all four new service methods and nil API preserve actual labels.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionAdmissionUsesOriginalSub400PolicyAndKeepsNativeRejections`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionServiceMethodsPreserveNativeBodiesCodesAndNilAPILabels`.
- Accepted Read/Close, captured-source drift and custom cancellation retain actual Applied and every joined cause, with Completed=false and an independently owned ResponseError. Body closes once and accepted IO errors invoke no caller retry or SDK replay; proof storage never aliases the caller response or another result.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsAcceptedFaultsKeepActualProofAndAllCauses`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- Live native401 reauthentication and bounded503 caller retry retain exact owned null body, captured ordinary headers, selected source and operation version. Body/decoder/retention changes, version conflicts or source drift stop before resend while original/callback/custom cancellation causes remain reachable.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionLiveReauthAndBoundedRetryKeepLiteralNullBody`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionRetryMutationStopsBodyVersionAndSourceBeforeResend`.
- Every physical POST including same-target307 redirects verifies exact method/path/origin, body bytes/framing and modern/legacy version policy. Changed body/framing/header/target stops before second transport attempt. Physical body Read/Close and header faults remain sticky after caller repair, close inspected body once and preserve actual causes.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionPhysicalRedirectMutationRemainsStickyAfterRepair`.

Python/Go differences:

- Go takes safe literal UTF-8 ID only. Source accepts mutable Volume/VolumeDetail/Resource/dict/Munch inputs through _get_resource with constructor/to_dict/descriptor/computed location side effects and Python urljoin coercion. Go skips those identities/side effects and name lookup deliberately; this does not classify separate Resource declarations.
- Source Proxy and Resource methods return None without translating response or updating local fields. Go adds independently owned Microversion, Discovery, Applied and Completed acknowledgement with literal input VolumeID. No semantic server-state confirmation, wait, follow-up GET or rollback is added.
- Volume uses class ceiling3.71 while Backup import keeps3.64 through the shared Cinder policy. Selected nonempty version is forwarded literally; Keystoneauth Session canonicalization, dynamic nonstring defaults, full Adapter cache/link/envelope/header discovery and deployment HTTP behavior are outside this owned Go mapping.
- Finite same-origin body-only first usable v3 discovery, typed advertisement shapes/status and clean concrete404/405 fallback are explicit mapping policy. Accepted malformed discovery is terminal with current proof rather than reproducing every caught Keystoneauth DiscoveryFailure/None path or endpoint-data exception order.
- Utility tuple selection retains maximum-before-optional-minimum, length/order and missing bounds. Shared arbitrary-precision Unicode16 decimal/underscore string parsing and latest handling are bounded Go behavior; exact installed CPython digit limits, nonstring input normalization and pinned dependency runtime are not claimed.
- Owned finite HTTP100..399 policy preserves opaque action bytes and admitted IO/source/context proof separately from native>=400 rejection. Native discarded rejected POST body Read/Close faults remain a documented limitation; mutable OkCodes cannot widen SDK acknowledgement. No response model conversion is introduced.
- Fixed source/method/route/body/framing and operation microversion ownership constrain mutable native retry/redirect hooks while live authentication and bounded caller retry remain. Native Reserve/Unreserve/BeginDetaching keep {} and narrower success codes; new null action APIs remain separate.
- Only the actual pinned v3 Proxy declaration is mapped here. Native/v2/Resource/inherited targets and direct bootable/readonly setters remain independent. Source hashes, static review, Go compile and offline HTTP/race tests do not establish authenticated OpenStack or Python SDK runtime parity or full SDK completion.

### `python:block_storage/v3/begin_volume_detaching`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.BeginVolumeDetaching`
- `github.com/JSYoo5B/go-openstacksdk.Connection.BeginVolumeDetaching`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.BeginVolumeDetaching`

- BeginVolumeDetaching sends exactly {"os-begin_detaching":null}. Use one explicit safe nonempty UTF-8 volume ID and the exact pinned null action through queryless POST volumes/{ID}/action. Unicode literal IDs are preserved, and VolumeDetail never inserts detail into the route. There is no UUID-only restriction, lookup, Resource conversion, status enum, options/defaults, builder, wait, refresh or rollback.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsPostSourceNullBodiesAndAcceptOpaqueHTTPReplies`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsRejectInvalidIDAndSourceBeforeAnyHTTP`.
- Direct calls capture the Cinder service and validate context/role/provider/endpoint/resource/header before ID admission. Connection checks context/provider/ID before selecting only cached Cinder v3, without CurrentLocation or recorded scope consumption. Getter errors retain the actual outer operation and custom cancellation causes before any request or fabricated proof.
  Tests: `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsUseOnlyCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsPreflightNeverSelectsCinder`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsGetterErrorsKeepOperationAndCancellation`.
- Selected nonempty microversion wins literally without discovery/cap. Otherwise operation-owned finite negotiation uses the Volume3.71 ceiling and server maximum before optional minimum, with tuple length/order retained; minimum above ceiling omits version and missing maximum ignores unused minimum. Original client/cache stays unchanged, each call owns its selected policy, and independent Backup import retains3.64 on the same server advertisement.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsSelectedVersionKeepsOwnedHeadersAndLiveNativeAuthentication`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsOwnDiscoveryPerCallAndKeepStageProof`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentStringAndHeaderCases`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentMaximumMinimumCases`.
- Finite discovery preserves captured encoded reverse-proxy prefix, admits200/300 supported body forms and first usable v3, and keeps admitted pages separate from action Applied. Only clean concrete native404/405 permits unavailable fallback. Accepted malformed/null/invalid-UTF8 or other discovery failures stop before action, with current ResponseError proof; later native rejection cannot borrow an earlier admitted page as current error proof.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsDiscoveryFailuresRetainCurrentStageAndNeverBorrowProof`.
- Action replies are opaque finite HTTP100..399 acknowledgements including empty, malformed and invalid-UTF8 bytes. Applied owns actual body/header/status, independently of separate calls or error proof. Completed means accepted Read/Close/source validation succeeded, never a server status postcondition. Original-policy>=400 and expanded native OkCodes retain native rejected evidence without Applied or Completed. Existing native Reserve/Unreserve/BeginDetaching keep their {} body and narrower status policy; all four new service methods and nil API preserve actual labels.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionAdmissionUsesOriginalSub400PolicyAndKeepsNativeRejections`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionServiceMethodsPreserveNativeBodiesCodesAndNilAPILabels`.
- Accepted Read/Close, captured-source drift and custom cancellation retain actual Applied and every joined cause, with Completed=false and an independently owned ResponseError. Body closes once and accepted IO errors invoke no caller retry or SDK replay; proof storage never aliases the caller response or another result.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsAcceptedFaultsKeepActualProofAndAllCauses`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- Live native401 reauthentication and bounded503 caller retry retain exact owned null body, captured ordinary headers, selected source and operation version. Body/decoder/retention changes, version conflicts or source drift stop before resend while original/callback/custom cancellation causes remain reachable.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionLiveReauthAndBoundedRetryKeepLiteralNullBody`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionRetryMutationStopsBodyVersionAndSourceBeforeResend`.
- Every physical POST including same-target307 redirects verifies exact method/path/origin, body bytes/framing and modern/legacy version policy. Changed body/framing/header/target stops before second transport attempt. Physical body Read/Close and header faults remain sticky after caller repair, close inspected body once and preserve actual causes.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionPhysicalRedirectMutationRemainsStickyAfterRepair`.

Python/Go differences:

- Go takes safe literal UTF-8 ID only. Source accepts mutable Volume/VolumeDetail/Resource/dict/Munch inputs through _get_resource with constructor/to_dict/descriptor/computed location side effects and Python urljoin coercion. Go skips those identities/side effects and name lookup deliberately; this does not classify separate Resource declarations.
- Source Proxy and Resource methods return None without translating response or updating local fields. Go adds independently owned Microversion, Discovery, Applied and Completed acknowledgement with literal input VolumeID. No semantic server-state confirmation, wait, follow-up GET or rollback is added.
- Volume uses class ceiling3.71 while Backup import keeps3.64 through the shared Cinder policy. Selected nonempty version is forwarded literally; Keystoneauth Session canonicalization, dynamic nonstring defaults, full Adapter cache/link/envelope/header discovery and deployment HTTP behavior are outside this owned Go mapping.
- Finite same-origin body-only first usable v3 discovery, typed advertisement shapes/status and clean concrete404/405 fallback are explicit mapping policy. Accepted malformed discovery is terminal with current proof rather than reproducing every caught Keystoneauth DiscoveryFailure/None path or endpoint-data exception order.
- Utility tuple selection retains maximum-before-optional-minimum, length/order and missing bounds. Shared arbitrary-precision Unicode16 decimal/underscore string parsing and latest handling are bounded Go behavior; exact installed CPython digit limits, nonstring input normalization and pinned dependency runtime are not claimed.
- Owned finite HTTP100..399 policy preserves opaque action bytes and admitted IO/source/context proof separately from native>=400 rejection. Native discarded rejected POST body Read/Close faults remain a documented limitation; mutable OkCodes cannot widen SDK acknowledgement. No response model conversion is introduced.
- Fixed source/method/route/body/framing and operation microversion ownership constrain mutable native retry/redirect hooks while live authentication and bounded caller retry remain. Native Reserve/Unreserve/BeginDetaching keep {} and narrower success codes; new null action APIs remain separate.
- Only the actual pinned v3 Proxy declaration is mapped here. Native/v2/Resource/inherited targets and direct bootable/readonly setters remain independent. Source hashes, static review, Go compile and offline HTTP/race tests do not establish authenticated OpenStack or Python SDK runtime parity or full SDK completion.

### `python:block_storage/v3/abort_volume_detaching`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.AbortVolumeDetaching`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AbortVolumeDetaching`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.AbortVolumeDetaching`

- AbortVolumeDetaching sends exactly {"os-roll_detaching":null}. Use one explicit safe nonempty UTF-8 volume ID and the exact pinned null action through queryless POST volumes/{ID}/action. Unicode literal IDs are preserved, and VolumeDetail never inserts detail into the route. There is no UUID-only restriction, lookup, Resource conversion, status enum, options/defaults, builder, wait, refresh or rollback.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsPostSourceNullBodiesAndAcceptOpaqueHTTPReplies`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsRejectInvalidIDAndSourceBeforeAnyHTTP`.
- Direct calls capture the Cinder service and validate context/role/provider/endpoint/resource/header before ID admission. Connection checks context/provider/ID before selecting only cached Cinder v3, without CurrentLocation or recorded scope consumption. Getter errors retain the actual outer operation and custom cancellation causes before any request or fabricated proof.
  Tests: `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsUseOnlyCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsPreflightNeverSelectsCinder`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsGetterErrorsKeepOperationAndCancellation`.
- Selected nonempty microversion wins literally without discovery/cap. Otherwise operation-owned finite negotiation uses the Volume3.71 ceiling and server maximum before optional minimum, with tuple length/order retained; minimum above ceiling omits version and missing maximum ignores unused minimum. Original client/cache stays unchanged, each call owns its selected policy, and independent Backup import retains3.64 on the same server advertisement.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsSelectedVersionKeepsOwnedHeadersAndLiveNativeAuthentication`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`, `connectiontest/connection_volume_state_actions_test.go:TestConnectionVolumeStateActionsOwnDiscoveryPerCallAndKeepStageProof`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentStringAndHeaderCases`, `internal/cloudsnapshot/backup_import_version_test.go:TestBackupImportVersionIndependentMaximumMinimumCases`.
- Finite discovery preserves captured encoded reverse-proxy prefix, admits200/300 supported body forms and first usable v3, and keeps admitted pages separate from action Applied. Only clean concrete native404/405 permits unavailable fallback. Accepted malformed/null/invalid-UTF8 or other discovery failures stop before action, with current ResponseError proof; later native rejection cannot borrow an earlier admitted page as current error proof.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsDiscoveryFailuresRetainCurrentStageAndNeverBorrowProof`.
- Action replies are opaque finite HTTP100..399 acknowledgements including empty, malformed and invalid-UTF8 bytes. Applied owns actual body/header/status, independently of separate calls or error proof. Completed means accepted Read/Close/source validation succeeded, never a server status postcondition. Original-policy>=400 and expanded native OkCodes retain native rejected evidence without Applied or Completed. Existing native Reserve/Unreserve/BeginDetaching keep their {} body and narrower status policy; all four new service methods and nil API preserve actual labels.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionAdmissionUsesOriginalSub400PolicyAndKeepsNativeRejections`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionServiceMethodsPreserveNativeBodiesCodesAndNilAPILabels`.
- Accepted Read/Close, captured-source drift and custom cancellation retain actual Applied and every joined cause, with Completed=false and an independently owned ResponseError. Body closes once and accepted IO errors invoke no caller retry or SDK replay; proof storage never aliases the caller response or another result.
  Tests: `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsAcceptedFaultsKeepActualProofAndAllCauses`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- Live native401 reauthentication and bounded503 caller retry retain exact owned null body, captured ordinary headers, selected source and operation version. Body/decoder/retention changes, version conflicts or source drift stop before resend while original/callback/custom cancellation causes remain reachable.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionLiveReauthAndBoundedRetryKeepLiteralNullBody`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionRetryMutationStopsBodyVersionAndSourceBeforeResend`.
- Every physical POST including same-target307 redirects verifies exact method/path/origin, body bytes/framing and modern/legacy version policy. Changed body/framing/header/target stops before second transport attempt. Physical body Read/Close and header faults remain sticky after caller repair, close inspected body once and preserve actual causes.
  Tests: `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionPhysicalRedirectMutationRemainsStickyAfterRepair`.

Python/Go differences:

- Go takes safe literal UTF-8 ID only. Source accepts mutable Volume/VolumeDetail/Resource/dict/Munch inputs through _get_resource with constructor/to_dict/descriptor/computed location side effects and Python urljoin coercion. Go skips those identities/side effects and name lookup deliberately; this does not classify separate Resource declarations.
- Source Proxy and Resource methods return None without translating response or updating local fields. Go adds independently owned Microversion, Discovery, Applied and Completed acknowledgement with literal input VolumeID. No semantic server-state confirmation, wait, follow-up GET or rollback is added.
- Volume uses class ceiling3.71 while Backup import keeps3.64 through the shared Cinder policy. Selected nonempty version is forwarded literally; Keystoneauth Session canonicalization, dynamic nonstring defaults, full Adapter cache/link/envelope/header discovery and deployment HTTP behavior are outside this owned Go mapping.
- Finite same-origin body-only first usable v3 discovery, typed advertisement shapes/status and clean concrete404/405 fallback are explicit mapping policy. Accepted malformed discovery is terminal with current proof rather than reproducing every caught Keystoneauth DiscoveryFailure/None path or endpoint-data exception order.
- Utility tuple selection retains maximum-before-optional-minimum, length/order and missing bounds. Shared arbitrary-precision Unicode16 decimal/underscore string parsing and latest handling are bounded Go behavior; exact installed CPython digit limits, nonstring input normalization and pinned dependency runtime are not claimed.
- Owned finite HTTP100..399 policy preserves opaque action bytes and admitted IO/source/context proof separately from native>=400 rejection. Native discarded rejected POST body Read/Close faults remain a documented limitation; mutable OkCodes cannot widen SDK acknowledgement. No response model conversion is introduced.
- Fixed source/method/route/body/framing and operation microversion ownership constrain mutable native retry/redirect hooks while live authentication and bounded caller retry remain. Native Reserve/Unreserve/BeginDetaching keep {} and narrower success codes; new null action APIs remain separate.
- Only the actual pinned v3 Proxy declaration is mapped here. Native/v2/Resource/inherited targets and direct bootable/readonly setters remain independent. Source hashes, static review, Go compile and offline HTTP/race tests do not establish authenticated OpenStack or Python SDK runtime parity or full SDK completion.

### `python:block_storage/v3/set_volume_bootable_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SetVolumeBootableStatus`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SetVolumeBootableStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SetVolumeBootableStatus`

- SetVolumeBootableStatus requires a bool and sends exactly {"os-set_bootable":{"bootable":value}}, including false, through the direct package, Connection and v3 service APIs without a default.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`.
- Use one safe explicit ID and fixed volumes/action route without name resolution.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsPreflightAndGetterErrorsKeepCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectPreflightPrecedesReadonlyOriginalsAndGuardsCapturedSource`.
- Selected version wins without cap; empty selection negotiates operation-owned Volume ceiling3.71.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsKeepNegotiationAndAppliedStagesSeparate`, `blockstorage/volume_flags_test.go:TestVolumeFlagsSelectedLiteralAndDiscoveryUseOwnedVolumeMicroversionPolicy`.
- Admitted response is opaque acknowledgement; no model refresh or server state assertion.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsKeepNegotiationAndAppliedStagesSeparate`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`, `blockstorage/volume_flags_test.go:TestVolumeFlagsNativeRejectionsAndExpandedStatusNeverBecomeAcknowledgements`.
- Accepted Read/Close/source/cancellation failures retain current Applied and every cause, without replay.
  Tests: `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`.
- Connection uses cached Cinder v3 and preserves typed preflight/getter causes without location work.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsPreflightAndGetterErrorsKeepCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.
- Exact body and version ownership reuse common scoped POST policy.
  Tests: `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsNativeRejectionsAndExpandedStatusNeverBecomeAcknowledgements`.
- New required-bool API remains independent of native200-only API and existing cloud defaultTrue lookup helper.
  Tests: `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.

Python/Go differences:

- Proxy requires bootable; Resource defaultTrue does not create a Proxy/Go default.
- Go bool narrows dynamic Python value domain; Source forwards values without descriptor BoolStr coercion.
- Source accepts ID/Resource/dict/Munch/None through local _get_resource; safe UTF-8 single-segment ID Go input excludes those dynamic constructor paths and unsafe IDs.
- No connected/disconnected Volume conversion, location computation, descriptor normalization or constructor warnings are claimed.
- Finite discovery/first usable v3/max-before-min handling is a bounded SDK policy, not complete Keystoneauth endpoint discovery/cache/links/status-priority parity.
- Missing maximum yields noMV; malformed accepted discovery is terminal evidence; clean direct404/405 fallback only.
- Current official Keystoneauth normalization reference is not a verified pinned dependency runtime.
- Source methods return None; Go deliberately returns owned Result and local completion evidence.
- Go explicit100..399 admission is a bounded HTTP mapping of Source status<400.
- SDK publishes response/error proof and local Completedfalse instead of Source None/raised error alone.
- Current new flag case covers joined Read+Close+source+customcancel for bootableFalse with Close1 and native retry0.
- Connection and typed service wrappers are SDK surfaces; no new Python cloud method or v2 promotion is inferred.
- Underlying Python Adapter auth/retry/redirect behavior is dependency-owned; SDK imposes stronger sticky fixed scope/body/framing/header ownership.
- New bootableFalse native503 retry with expanded400 rejection has an actual public assertion; dedicated bootable changed-body/version/307 matrix is not a new flag-specific case. Shared state transport remains explicit common-policy reuse only.
- Native BootableOpts zero valueFalse serializes; nativeSetBootable accepts200 only; new acknowledgement mapping accepts wider Source sub400 codes.
- New service test independently exercises native200/error-only, new Source203 acknowledgement, nilAPI labels, and existing cloud lookup/defaultTrue. It does not promote the native or cloud targets.
- Only this actual pinned v3 Proxy declaration is mapped. Resource/inherited/v2/native targets, authenticated OpenStack/Python SDK runtime and full SDK completion are not classified by these offline Go contracts. Rejected POST body IO discarded by native Gophercloud remains a limitation.

### `python:block_storage/v3/set_volume_readonly`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SetVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SetVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SetVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeReadonlyOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeReadonlyOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeReadonly`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeReadonlyOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeReadonlyOptions`

- SetVolumeReadonly sends exactly {"os-update_readonly_flag":{"readonly":value}}. Omitted options or a nil Readonly field default to true; WithVolumeReadonly(false) and a nonnil false pointer retain explicit false.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyOptionsDefaultsFalseReplacementAndFactoryCopies`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`.
- Complete replacement, order and reusable factories own caller and returned bool pointers.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyOptionsDefaultsFalseReplacementAndFactoryCopies`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`.
- Original callbacks run in captured order and cannot replace callback slice or retained prior config.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyOriginalsOwnCallbackSliceAndRetainedPointersThroughHTTP`.
- Service-free Prepare preserves context/callback causes and stops later originals on failure.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyPrepareStopsNilContextCancellationNilCallbackAndErrors`, `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.
- Direct source capture and safe-ID validation precede originals; observed source violations are sticky.
  Tests: `blockstorage/volume_boolean_options_test.go:TestVolumeReadonlyDirectPreflightAndStickySourceErrorsStopLaterOriginals`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectPreflightPrecedesReadonlyOriginalsAndGuardsCapturedSource`.
- Connection originals run once before cached getter, then only owned config factory reaches core.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeReadonlyPreparesOriginalsOnceBeforeGetter`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsPreflightAndGetterErrorsKeepCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.
- Shared Volume3.71 negotiation and opaque evidence keep Discovery separate from Applied/Completed.
  Tests: `connectiontest/connection_volume_flags_test.go:TestConnectionVolumeFlagsKeepNegotiationAndAppliedStagesSeparate`, `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup`, `blockstorage/volume_flags_test.go:TestVolumeFlagsNativeRejectionsAndExpandedStatusNeverBecomeAcknowledgements`, `blockstorage/volume_flags_test.go:TestVolumeFlagsSelectedLiteralAndDiscoveryUseOwnedVolumeMicroversionPolicy`.
- Owned false request survives bounded native retry and rejects changed bool before resend while preserving native evidence.
  Tests: `blockstorage/volume_flag_transport_test.go:TestVolumeReadonlyRetryPreservesFalseAndRejectsChangedFlag`, `blockstorage/volume_flag_transport_test.go:TestVolumeFlagsAcceptedFaultsKeepOpaqueProofAndJoinedCauses`, `blockstorage/volume_flags_test.go:TestVolumeFlagsNativeRejectionsAndExpandedStatusNeverBecomeAcknowledgements`, `blockstorage/volume_flags_test.go:TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent`.

Python/Go differences:

- Proxy defaultsTrue; Resource requires readonly.
- Go bool pointer omits/defaults nil; Source explicit dynamic None would forward JSONnull rather than become defaultTrue.
- Options, full replacement and pointer ownership are SDK additions to a positional Source bool.
- Source has no option callback layer; owned ordinary headers plus live auth are SDK policy.
- Service-free preparation and nil callback validation are SDK additions; no HTTP/catalog selection occurs in Prepare.
- Source Resource/dict/Munch/None constructor/model/location paths are excluded by typed safe-ID input.
- SDK observes source/context around callback boundaries; it does not claim to detect a mutation restored entirely inside a single callback.
- SDK Connection sequencing is distinct from direct Source capture sequencing; Prepare runs before selecting service.
- No CurrentLocation, model normalization, lookup or v2 promotion is introduced.
- Source None becomes SDK owned Result; no refreshed Volume/read-only/status model is fabricated.
- Finite discovery and explicit100..399 admission remain bounded mappings; actual public tests bind per-target3.71/belowcap/minabovecap/missingmax and native/expanded400 rejection.
- Current readonly-specific retry covers one503retry unchangedFalse and changedTrue rejection with original503method/URL/body/header via errors.As.
- Exhaustive framing/version/307/auth variants are shared policy reuse, not additional new readonly flag cases.
- Native AttachMode ReadOnly ro is unrelated; no native readonly action was invented.
- Only this actual pinned v3 Proxy declaration is mapped. Resource/inherited/v2/native targets, authenticated OpenStack/Python SDK runtime and full SDK completion are not classified by these offline Go contracts. Rejected POST body IO discarded by native Gophercloud remains a limitation.

### `python:block_storage/v3/extend_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ExtendVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ExtendVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ExtendVolume`

- ExtendVolume requires a machine integer and sends os-extend.new_size exactly, including zero and negative values. The typed value is encoded without the native float64 request-builder roundtrip; the 64-bit-only independent case retains9007199254740993. No local positivity or existing-size comparison is imposed.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Direct calls capture context and Cinder role/provider/route/header policy before validating one safe explicit volume ID and making queryless POST volumes/{ID}/action. No name/Type lookup, Resource conversion, location work or wait occurs; unsafe ID, nil context/client, wrong role and reserved auth fail locally with the actual operation and no fabricated response proof.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectPreflightValidatesOnlySelectedSourceAndUTF8BodyText`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizePreflightAndGetterErrorsKeepCauses`.
- Connection uses only cached Cinder v3 and ignores invalid recorded cloud location. Context/Connection/provider/ID preflight and getter failures retain operation/resource identity and custom cancellation causes before any HTTP.
  Tests: `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizePreflightAndGetterErrorsKeepCauses`.
- Selected3.1,3.80 or latest is forwarded literally without discovery, ceiling or minimum gate. Empty selection derives an owned Volume3.71 ceiling, respects lower maximum3.70 and omits version when minimum3.72 exceeds the ceiling. Discovery proof is separate from Applied and selection remains per-call without changing cached client version.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeSelectedLiteralAndDiscoveryKeepActionPayloadAndOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeNegotiationKeepsAcceptedStagesSeparate`.
- Owned HTTP100..399 admission retains opaque action bytes without parsing. Current extend-specific cases use200/203 and302 in the64-bit precision case, and populate independently owned Applied plus literal input VolumeID. Completed is local helper acknowledgement and adds no server size/status confirmation or follow-up request.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Native400 and one503retry with expanded native400 acceptance retain actual native status/body/header while original SDK admission rejects400: no Applied, ResponseError or Completed is fabricated.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeOriginalHTTPPolicyKeepsRejectedNativeEvidence`.
- Accepted Read and Close faults joined with observed Source drift and custom cancellation retain current Applied plus independently owned ResponseError and every original cause. Body closes once, Completed is false, and accepted faults invoke no native retry or SDK replay.
  Tests: `blockstorage/volume_resize_transport_test.go:TestVolumeResizeAcceptedFaultsKeepActualOpaqueProofAndEveryCause`.
- New v3 service ExtendVolume uses the required typed size and opaque Source acknowledgement while existing ExtendSize retains its current numeric-zero builder rule and202-only error contract. Nil API produces the actual ExtendVolume label without HTTP or response proof.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`.

Python/Go differences:

- Source size annotations are not runtime conversion/validation: arbitrary-precision integers, bool, null, strings and containers can be forwarded dynamically. Go deliberately narrows to the platform machine int and includes zero/negative; the precision case is guarded for64-bit platforms.
- Native numeric-required zero exemption and its request-builder float conversion are observed current local/cache snapshot behavior, not a claim about a fresh upstream distribution. Existing ExtendSize is unchanged and unpromoted.
- New extend cases exercise accepted joined faults, original-policy rejection and native expanded503retry. Dedicated extend401 reauth, changed-body/version and307 matrices remain explicit shared Cinder policy reuse rather than additional extend-specific test cases.
- Go requires one explicit safe UTF-8 single-segment volume ID. Source accepts Volume/VolumeDetail/Resource/dict/Munch/None through local _get_resource with constructor, descriptor, location and mutable identity effects; Go skips those paths and custom urljoin coercion without classifying their separate declarations.
- Source returns None and does not refresh Volume fields. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement; Completed never confirms size/type/server status. No lookup, CurrentLocation, model conversion, poll, wait or automatic undo is added.
- Selected nonempty microversion is literal and has no additional API-minimum gate. Empty selection reuses bounded operation-owned Volume3.71 discovery, maximum before optional minimum, clean concrete native404/405 fallback and current accepted proof. Full Keystoneauth discovery/cache/link/envelope/status ordering and session canonicalization are outside this mapping; official normalization reference is not a verified pinned dependency runtime.
- Finite HTTP100..399 is the owned mapping of Source status<400. Action bytes remain opaque; native>=400 or expanded OkCodes cannot become an SDK acknowledgement. Native discarded rejected POST body IO remains a limitation.
- Captured source and exact method/route/body/framing/version ownership constrain mutable native retry and redirect hooks. Mutations restored wholly inside one callback are not detectable. Auth/retry/redirect dependency behavior and authenticated OpenStack/Python runtime parity are not established by these offline contracts.
- Only this actual pinned v3 Proxy declaration is a verified Go mapping. Native/resource/inherited/v2/cloud targets and shared dependencies remain independently classified; no full SDK completion or automatic support promotion is inferred.

### `python:block_storage/v3/retype_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.RetypeVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.RetypeVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.RetypeVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeRetypeMigrationPolicy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeRetypeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeRetypeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeRetypeMigrationPolicy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeRetypeOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeRetypeOptions`

- RetypeVolume always includes literal new_type. Required UTF-8 text may be empty or contain whitespace, slash, query/percent/fragment characters or controls; it is neither route-validated nor resolved by Type GET. Nil MigrationPolicy defaults to Proxy never, explicit empty omits migration_policy, and arbitrary valid policy is accepted without enum validation.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeOptionsDefaultsEmptyReplacementAndFactoriesOwnValues`.
- Complete replacement and later options determine final policy. Factories copy caller pointers when created and every invocation/result owns a fresh pointer; callbacks execute a copied original slice in order with copied intermediate configs, so replacement of a later caller callback or mutation of retained prior pointers cannot alter the request. Captured ordinary headers stay original while provider authentication is live.
  Tests: `blockstorage/volume_resize_options_test.go:TestVolumeResizeOptionsDefaultsEmptyReplacementAndFactoriesOwnValues`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeOriginalsOwnOrderCallbackSliceAndRetainedPointersThroughHTTP`.
- Service-free Prepare runs originals once without selecting a service or doing HTTP. Nil context/options and callback/context failures stop later originals, retain all callback and custom cancellation causes, and return zero config. Migration policy UTF-8 is checked only after originals, permitting an invalid earlier value to be replaced by a valid final value before any action.
  Tests: `blockstorage/volume_resize_options_test.go:TestVolumeResizePreparePreservesCausesAndValidatesOnlyFinalRetypePolicy`.
- Direct calls capture Cinder Source, validate safe volume ID, then validate newType UTF-8 before originals. Source/context is guarded around originals; an observed route/source fault remains terminal and prevents later repair callbacks or HTTP while keeping joined callback and cancellation causes.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectPreflightValidatesOnlySelectedSourceAndUTF8BodyText`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeDirectPreflightAndStickySourcesStopOriginalsAndHTTP`.
- Connection preflight precedes one original Prepare; final migration policy and newType are checked before ID admission and only cached Cinder v3 selection. The getter cannot mutate a retained original config into the action; only an owned full-policy factory reaches core. Location is ignored and preflight/getter errors preserve actual operation and cancellation causes.
  Tests: `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeOriginalsRunOnceBeforeGetterWithOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizePreflightAndGetterErrorsKeepCauses`.
- Selected literal3.1/3.80/latest bypasses discovery and class ceiling without a new minimum gate. Otherwise finite per-call selection uses Volume3.71, lower maximum or no version for minimum above ceiling; cached source is unchanged and accepted discovery pages remain separate from action Applied, including terminal malformed discovery evidence.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeSelectedLiteralAndDiscoveryKeepActionPayloadAndOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeNegotiationKeepsAcceptedStagesSeparate`.
- Opaque200/203/302/399 replies own raw Applied and local Completed independently across calls. Native400 and503retry expanded400 stay native rejected evidence with no Applied or success proof; no response JSON or refreshed Volume type/status model is created.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_test.go:TestVolumeResizeOriginalHTTPPolicyKeepsRejectedNativeEvidence`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Accepted Read/Close/source/custom cancellation retains current opaque Applied, independent ResponseError, every joined cause, close-once and Completed=false without replay or caller retry.
  Tests: `blockstorage/volume_resize_transport_test.go:TestVolumeResizeAcceptedFaultsKeepActualOpaqueProofAndEveryCause`.
- One native401 reauth and one bounded503retry use live token with original captured ordinary headers and exact owned never policy despite mutation of retained option pointer. A retry changing migration_policy is stopped before third transport and retains original503method/URL/body/header; originals are never rerun.
  Tests: `blockstorage/volume_resize_transport_test.go:TestVolumeRetypeReauthRetryKeepsOwnedPolicyAndRejectsChangedBody`.
- New service RetypeVolume admits empty literal newType, defaultnever and explicit empty omission, preserving actual labels on nil API. Existing ChangeType still requires nonempty NewType, omits empty migration policy and accepts202 only; those native builders and targets are unchanged.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`.

Python/Go differences:

- Source accepts a Type Resource through Resource._get_id, or forwards arbitrary non-Resource values including null/bool/container domains. Go newType and MigrationPolicy are typed UTF-8 strings, deliberately omit constructor/Resource identity and preserve valid literal body text without Type lookup.
- Proxy defaultnever differs from Resource.retype defaultNone. Go nil pointer means Proxy defaultnever; explicit empty string models omitted field, rather than reproducing explicit Python None/False/0 dynamic inputs.
- Factories, options, service-free preparation, copied callback slices/configs, zero-on-error and direct versus Connection ordering are SDK-owned contracts absent from positional Python Source.
- Current retype-specific native401/503 case covers retained policy mutation, live auth and changed body rejection. Exhaustive version/framing/307 matrices remain explicit shared Cinder transport reuse, not new retype-specific cases.
- Native ChangeType required nonempty string and202-only policy stay independent of the new Source mapping; enum constants are not runtime validation and no native target is promoted.
- Go requires one explicit safe UTF-8 single-segment volume ID. Source accepts Volume/VolumeDetail/Resource/dict/Munch/None through local _get_resource with constructor, descriptor, location and mutable identity effects; Go skips those paths and custom urljoin coercion without classifying their separate declarations.
- Source returns None and does not refresh Volume fields. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement; Completed never confirms size/type/server status. No lookup, CurrentLocation, model conversion, poll, wait or automatic undo is added.
- Selected nonempty microversion is literal and has no additional API-minimum gate. Empty selection reuses bounded operation-owned Volume3.71 discovery, maximum before optional minimum, clean concrete native404/405 fallback and current accepted proof. Full Keystoneauth discovery/cache/link/envelope/status ordering and session canonicalization are outside this mapping; official normalization reference is not a verified pinned dependency runtime.
- Finite HTTP100..399 is the owned mapping of Source status<400. Action bytes remain opaque; native>=400 or expanded OkCodes cannot become an SDK acknowledgement. Native discarded rejected POST body IO remains a limitation.
- Captured source and exact method/route/body/framing/version ownership constrain mutable native retry and redirect hooks. Mutations restored wholly inside one callback are not detectable. Auth/retry/redirect dependency behavior and authenticated OpenStack/Python runtime parity are not established by these offline contracts.
- Only this actual pinned v3 Proxy declaration is a verified Go mapping. Native/resource/inherited/v2/cloud targets and shared dependencies remain independently classified; no full SDK completion or automatic support promotion is inferred.

### `python:block_storage/v3/complete_volume_extend`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.CompleteVolumeExtend`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CompleteVolumeExtend`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.CompleteVolumeExtend`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeExtendCompletionError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeExtendCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeExtendCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeExtendCompletionError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeExtendCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeExtendCompletionOptions`

- CompleteVolumeExtend sends os-extend_volume_completion.error on every action. Omitted options/nil Error default to false; explicit false and true retain the supplied bool, and full empty replacement restores false. Error=true informs Cinder of an extend error; SDK performs no rollback or backend completion check.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeOptionsDefaultsEmptyReplacementAndFactoriesOwnValues`, `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Reusable full-policy factories copy caller input and own each returned bool. Later options win, original callback slice is snapshotted and intermediate/retained config pointers cannot alter later callbacks or the HTTP body; captured ordinary headers remain original while token is live.
  Tests: `blockstorage/volume_resize_options_test.go:TestVolumeResizeOptionsDefaultsEmptyReplacementAndFactoriesOwnValues`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeOriginalsOwnOrderCallbackSliceAndRetainedPointersThroughHTTP`.
- Service-free Prepare validates context and nil callbacks, retains callback/custom cancellation causes, stops later originals on failure and returns zero config. Defaults and returned pointers are owned, and no client selection or HTTP is performed.
  Tests: `blockstorage/volume_resize_options_test.go:TestVolumeResizePreparePreservesCausesAndValidatesOnlyFinalRetypePolicy`.
- Direct source capture and safe-ID admission precede originals. Source/context guards around originals make observed source drift and joined callback/cancellation failure terminal before later repair callbacks or HTTP, with no fabricated response proof.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectPreflightValidatesOnlySelectedSourceAndUTF8BodyText`, `blockstorage/volume_resize_options_test.go:TestVolumeResizeDirectPreflightAndStickySourcesStopOriginalsAndHTTP`.
- Connection preflights context/Connection/provider, runs originals once before safe-ID/getter selection, and passes only copied complete policy to core. Getter mutation of retained config cannot change true body; cached Cinder v3 ignores location, and preflight/getter failures retain operation and custom cancellation identities.
  Tests: `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeOriginalsRunOnceBeforeGetterWithOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizePreflightAndGetterErrorsKeepCauses`.
- Selected literal3.1/3.80/latest bypasses discovery/ceiling and no minimum gate is added. Empty selection uses finite per-call Volume3.71/lower maximum/noMV for high minimum; admitted discovery evidence and action Applied remain distinct, and cached service version stays unchanged.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeSelectedLiteralAndDiscoveryKeepActionPayloadAndOwnedPolicy`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeNegotiationKeepsAcceptedStagesSeparate`.
- Accepted200/203/302/399 opaque bytes populate owned Applied and local helper Completed without parsing or refreshing Volume state. Native400 and one503 expanded400 retry retain rejected native body/header and cannot become Applied or Completed.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults`, `blockstorage/volume_resize_test.go:TestVolumeResizeOriginalHTTPPolicyKeepsRejectedNativeEvidence`, `connectiontest/connection_volume_resize_test.go:TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation`.
- Accepted Read/Close faults, observed Source drift and custom cancellation retain every cause and current opaque Applied plus independent ResponseError. Close executes once, Completed=false, and no accepted-fault caller retry or replay occurs.
  Tests: `blockstorage/volume_resize_transport_test.go:TestVolumeResizeAcceptedFaultsKeepActualOpaqueProofAndEveryCause`.
- New service CompleteVolumeExtend sends defaultfalse or explicittrue and admits opaque acknowledgement; nil API preserves actual completion operation label without originals, request or proof. This helper does not wait for volume extension or invent a native completion API.
  Tests: `blockstorage/volume_resize_test.go:TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels`.

Python/Go differences:

- Source positional error annotation is not runtime bool conversion. Explicit dynamic None/nonbool values can be forwarded as JSON; Go optional *bool deliberately owns defaultFalse and narrows the domain to bool.
- Options, reusable full replacement, copied callback/config ownership, service-free preparation and Connection originals-before-getter ordering are SDK additions to the Source positional defaultFalse contract.
- The completion action is an acknowledgement, not a poll or server-state postcondition; error=true is Cinder input rather than SDK rollback. No native completion builder/API is invented.
- New completion cases cover accepted joined faults and original-policy rejection including expanded503retry. Dedicated completion401/changed-body/version/307 matrices remain explicit shared scoped Cinder policy reuse rather than new completion-specific cases.
- Go requires one explicit safe UTF-8 single-segment volume ID. Source accepts Volume/VolumeDetail/Resource/dict/Munch/None through local _get_resource with constructor, descriptor, location and mutable identity effects; Go skips those paths and custom urljoin coercion without classifying their separate declarations.
- Source returns None and does not refresh Volume fields. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement; Completed never confirms size/type/server status. No lookup, CurrentLocation, model conversion, poll, wait or automatic undo is added.
- Selected nonempty microversion is literal and has no additional API-minimum gate. Empty selection reuses bounded operation-owned Volume3.71 discovery, maximum before optional minimum, clean concrete native404/405 fallback and current accepted proof. Full Keystoneauth discovery/cache/link/envelope/status ordering and session canonicalization are outside this mapping; official normalization reference is not a verified pinned dependency runtime.
- Finite HTTP100..399 is the owned mapping of Source status<400. Action bytes remain opaque; native>=400 or expanded OkCodes cannot become an SDK acknowledgement. Native discarded rejected POST body IO remains a limitation.
- Captured source and exact method/route/body/framing/version ownership constrain mutable native retry and redirect hooks. Mutations restored wholly inside one callback are not detectable. Auth/retry/redirect dependency behavior and authenticated OpenStack/Python runtime parity are not established by these offline contracts.
- Only this actual pinned v3 Proxy declaration is a verified Go mapping. Native/resource/inherited/v2/cloud targets and shared dependencies remain independently classified; no full SDK completion or automatic support promotion is inferred.

### `python:block_storage/v3/reset_volume_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.ResetVolumeStatus`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetVolumeStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ResetVolumeStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeStatusResetOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeStatusResetStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeStatusResetAttachStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeStatusResetMigrationStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeStatusResetOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeStatusResetOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeStatusResetStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeStatusResetAttachStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeStatusResetMigrationStatus`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeStatusResetOptions`

- Reset defaults all three statuses to omitted. Nil and empty typed strings are omitted, all nonempty UTF-8 strings are sent independently without enum/default injection; empty options still POST {os-reset_status:{}}.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`.
- Reusable owned option factories/full replacement and service-free preparation preserve copied pointers, original callback slice and effective policy; no caller builder is needed.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationOriginalOptionsOwnFactoryPointersRetainedConfigsAndCallbackOrder`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`.
- Direct/source admission and Connection prepare-before-cached-Cinder ordering retain operation/context/getter faults before HTTP, without lookup or CurrentLocation.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsCaptureSourcesBeforeOriginalsAndStopBeforeRestoration`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsPreflightAndCallbackFailuresPreservePriorityAndCauses`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationPreflightAndGetterFailuresRetainCauses`.
- Selected literal/cap3.71 uses ordinary shared Cinder negotiation with no API minimum gate; opaque action proof and Completed acknowledgement remain distinct.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`.
- ResetVolumeStatus reaches the same applyRequired/Post accepted-fault policy whose existing Cinder regression retains actual Applied/all joined causes without completion or replay. New reset cases directly verify original native400/expanded503retry rejection keeps native evidence and cannot become Applied or Completed.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsKeepOriginalNativePolicyWhenRetryExpandsHTTPAdmission`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.
- New package/Connection/service ResetVolumeStatus methods share the exact typed action and preserve nil API operation labels without originals or HTTP. Existing native ResetStatus remains a separately inspected source/static API; its always-included status/202/error-only behavior is not asserted by this new service test.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`.

Python/Go differences:

- Source truthy status accepts arbitrary dynamic values; Go deliberately narrows to nullable UTF-8 strings and nonempty omission. Existing native required string body is not reused as Source-equivalent.
- Native Volume ResetStatus always-included status/202/error-only separation is established by current native/generated Source declarations and unchanged signature, not a newly executed native reset preservation scenario.
- Go takes one safe UTF-8 current volume ID and typed strings/bools rather than supplied mutable Volume/VolumeDetail/Resource/dict/Munch inputs and Source constructor/descriptor/location/urljoin side effects. Separate Resource/native/v2 declarations remain independent.
- Source returns None without response translation or refresh. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement. No lookup, CurrentLocation, asynchronous wait/poll, cleanup or undo is added.
- Finite call-owned discovery and typed advertisement/evidence policy are explicit Go mappings. Complete Keystoneauth cache/link/envelope/canonicalization/discovery exception ordering and authenticated backend behavior are not claimed. Empty Go selection cannot represent Python explicit default empty separately from None.
- Opaque finite100..399 action policy maps Source<400. Accepted body IO/source/context proof remains separate from native rejection; native discarded rejected POST body IO remains a limitation. Exact method/route/body/framing/version guards constrain native retry/redirect while live auth remains.
- New11 groups cover typed body/defaults, options/capture/preflight, conditional required discovery, original admission and facade/service behavior. Accepted action IO, ordinary cap details and complete reauth/307/framing matrices use declared shared Cinder engine reuse where applicable; no new operation-specific exhaustive transport matrix is claimed. Standalone Prepare wrappers are compiled and exercised through Connection preparation, not a separate new reusable-Prepare return mutation matrix.
- Only the actual pinned v3 Proxy target is reviewed here. Root offline local HTTP race tests, vet and exact documentation compilation passed. Static Source hashes and offline Go checks do not prove authenticated OpenStack/Python runtime parity or full SDK completion.

### `python:block_storage/v3/migrate_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.MigrateVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.MigrateVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.MigrateVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationHost`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationCluster`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationForceHostCopy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationLockVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeMigrationOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationHost`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationCluster`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationForceHostCopy`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationLockVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeMigrationOptions`

- Host/Cluster nil omission differs from explicit empty inclusion. False/default ForceHostCopy/LockVolume fields are omitted; true fields are included. No host/cluster resolver, requirement or mutual-exclusion check is added.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`.
- Cluster nonnil, including empty, requires advertised server min/max to support3.16 before action, even when a selected version exists; compatible selected same-major>=3.16 is additionally required. Failure prevents POST. The pinned Keystoneauth5.16.0 version_match has exact same-major/tuple semantics: global latest fails required3.16, while3.latest can pass the gate and keeps its literal POST policy.
  Tests: `blockstorage/volume_migration_test.go:TestMigrateVolumeRequiresAdvertisedSupportBeforeSelectedOrComputedVersion`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`.
- Required server range tests3.16 rather than selected literal; high compatible selected version stays literal above advertised maximum. Unselected calls select cap3.71 from the same captured advertisement; no original client version mutation or dependency cache emulation.
  Tests: `blockstorage/volume_migration_test.go:TestMigrateVolumeRequiresAdvertisedSupportBeforeSelectedOrComputedVersion`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`.
- Owned option preparation and original callback once/direct capture/Connection cached Cinder preserve caller-owned values and fixed payload across retry.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationOriginalOptionsOwnFactoryPointersRetainedConfigsAndCallbackOrder`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsCaptureSourcesBeforeOriginalsAndStopBeforeRestoration`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsKeepOriginalNativePolicyWhenRetryExpandsHTTPAdmission`.
- Discovery and action phases own separate current proof; malformed accepted discovery, missing/unsupported gate bounds and source/context faults prevent action and preserve actual admitted evidence.
  Tests: `blockstorage/volume_migration_test.go:TestMigrateVolumeRequiresAdvertisedSupportBeforeSelectedOrComputedVersion`, `blockstorage/volume_migration_test.go:TestMigrateVolumeFiniteDiscoveryKeepsCurrentProofAndNeverBorrowsRejectedPages`.
- New MigrateVolume cases verify opaque action acknowledgement and original native rejection through the shared action engine, with no asynchronous wait, host GET, volume refresh or rollback. Existing shared Cinder accepted Read/Close/source/custom-context regression verifies the common Post failure path; it is not a new cluster-action IO matrix.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsKeepOriginalNativePolicyWhenRetryExpandsHTTPAdmission`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.

Python/Go differences:

- Source host/cluster arbitrary non-None values and flags arbitrary truthy values are narrowed to typed nullable strings/bools; explicit cluster empty retains required gate.
- Source require calls endpoint getter and both server bounds before default normalization; Go shares one finite guarded probe with ordinary action version selection and explicitly omits full Keystoneauth cache/canonicalization/taxonomy. Python selected default empty versus None is not independently expressible in Go.
- Keystoneauth5.16.0 commit e759cf88c072d5bc7e21fc3fa884708ace9630af statically binds normalize_version_number/version_between/version_match. Source requires>=5.16.0; this does not execute installed dependency/runtime or promote Adapter cache/Session behavior. Bounds normalize up front, required3.16 range is tuple comparison and selected gate requires equal major plus candidate>=required. Literal selected POST header forwarding remains a declared Go-versus-Session canonicalization boundary.
- Go takes one safe UTF-8 current volume ID and typed strings/bools rather than supplied mutable Volume/VolumeDetail/Resource/dict/Munch inputs and Source constructor/descriptor/location/urljoin side effects. Separate Resource/native/v2 declarations remain independent.
- Source returns None without response translation or refresh. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement. No lookup, CurrentLocation, asynchronous wait/poll, cleanup or undo is added.
- Finite call-owned discovery and typed advertisement/evidence policy are explicit Go mappings. Complete Keystoneauth cache/link/envelope/canonicalization/discovery exception ordering and authenticated backend behavior are not claimed. Empty Go selection cannot represent Python explicit default empty separately from None.
- Opaque finite100..399 action policy maps Source<400. Accepted body IO/source/context proof remains separate from native rejection; native discarded rejected POST body IO remains a limitation. Exact method/route/body/framing/version guards constrain native retry/redirect while live auth remains.
- New11 groups cover typed body/defaults, options/capture/preflight, conditional required discovery, original admission and facade/service behavior. Accepted action IO, ordinary cap details and complete reauth/307/framing matrices use declared shared Cinder engine reuse where applicable; no new operation-specific exhaustive transport matrix is claimed. Standalone Prepare wrappers are compiled and exercised through Connection preparation, not a separate new reusable-Prepare return mutation matrix.
- Only the actual pinned v3 Proxy target is reviewed here. Root offline local HTTP race tests, vet and exact documentation compilation passed. Static Source hashes and offline Go checks do not prove authenticated OpenStack/Python runtime parity or full SDK completion.

### `python:block_storage/v3/complete_volume_migration`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.CompleteVolumeMigration`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CompleteVolumeMigration`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.CompleteVolumeMigration`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeMigrationCompletionError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeMigrationCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationCompletionOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeMigrationCompletionError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeMigrationCompletionOptions`

- Required newVolume is a literal UTF-8 body string, including empty/route-like text; it is not resolved or validated as the current route ID. new_volume and error are always sent.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`.
- Omitted/nil Error defaultsfalse, explicitfalse/true is always transmitted; reusable full policy factories own pointers and preserve callback once semantics.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationOriginalOptionsOwnFactoryPointersRetainedConfigsAndCallbackOrder`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`.
- Direct/current ID and Connection prepare-before-Cinder admission keep operation/context/getter errors with no new-volume GET or CurrentLocation.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsCaptureSourcesBeforeOriginalsAndStopBeforeRestoration`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsPreflightAndCallbackFailuresPreservePriorityAndCauses`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationPreflightAndGetterFailuresRetainCauses`.
- No API minimum gate is added; selected literal/ordinary cap3.71 selection, separate Discovery and opaque Applied are operation-owned.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`.
- Completed proves accepted body read/close/source helper completion only; Error=true informs Cinder, not SDK cleanup, migration completion, wait or undo.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`, `connectiontest/connection_volume_migration_test.go:TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery`.
- Package/Connection/service completion and nil service labels match the actual helper. New cases verify rejected action/native policy and preflight causes; the shared Post accepted IO/source/custom-context regression retains current proof/all causes without replay and is explicit reused engine evidence.
  Tests: `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsPreflightAndCallbackFailuresPreservePriorityAndCauses`, `blockstorage/volume_migration_test.go:TestVolumeMigrationActionsKeepOriginalNativePolicyWhenRetryExpandsHTTPAdmission`, `blockstorage/volume_migration_test.go:TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority`, `blockstorage/volume_state_action_transport_test.go:TestVolumeStateActionsAcceptedFaultsKeepAppliedAndEveryCauseWithoutCompletion`.

Python/Go differences:

- Source positional new_volume/error annotations do not perform coercion and allow dynamic null/nonbool values; Go deliberately uses required UTF-8 string and owned optionalbool.
- No native completion API is invented and no new-volume Resource constructor/_get_id shortcut or resolver is introduced.
- Go takes one safe UTF-8 current volume ID and typed strings/bools rather than supplied mutable Volume/VolumeDetail/Resource/dict/Munch inputs and Source constructor/descriptor/location/urljoin side effects. Separate Resource/native/v2 declarations remain independent.
- Source returns None without response translation or refresh. Go adds owned VolumeID, Microversion, Discovery, Applied and Completed acknowledgement. No lookup, CurrentLocation, asynchronous wait/poll, cleanup or undo is added.
- Finite call-owned discovery and typed advertisement/evidence policy are explicit Go mappings. Complete Keystoneauth cache/link/envelope/canonicalization/discovery exception ordering and authenticated backend behavior are not claimed. Empty Go selection cannot represent Python explicit default empty separately from None.
- Opaque finite100..399 action policy maps Source<400. Accepted body IO/source/context proof remains separate from native rejection; native discarded rejected POST body IO remains a limitation. Exact method/route/body/framing/version guards constrain native retry/redirect while live auth remains.
- New11 groups cover typed body/defaults, options/capture/preflight, conditional required discovery, original admission and facade/service behavior. Accepted action IO, ordinary cap details and complete reauth/307/framing matrices use declared shared Cinder engine reuse where applicable; no new operation-specific exhaustive transport matrix is claimed. Standalone Prepare wrappers are compiled and exercised through Connection preparation, not a separate new reusable-Prepare return mutation matrix.
- Only the actual pinned v3 Proxy target is reviewed here. Root offline local HTTP race tests, vet and exact documentation compilation passed. Static Source hashes and offline Go checks do not prove authenticated OpenStack/Python runtime parity or full SDK completion.

### `python:block_storage/v3/revert_volume_to_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.RevertVolumeToSnapshot`
- `github.com/JSYoo5B/go-openstacksdk.Connection.RevertVolumeToSnapshot`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.RevertVolumeToSnapshot`

- Revert uses a required literal UTF-8 snapshotID body string, including empty/control/path-like strings, with exact {revert:{snapshot_id:value}} and safe current volume action route. No snapshot/current-volume GET or name resolver.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`.
- Required3.40 server support is checked before action even when selected version exists: both advertisement bounds must contain3.40 and selected candidate must same-major>=3.40; compatible selected above server maximum remains raw literal. New revert range cases verify3.39/equal3.40/high selected and missing/range bounds; global latest/3.latest same-major comparisons are existing requireSupport engine regression plus actual fixed3.40 Source/code evidence.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestRevertVolumeToSnapshotRequiresAdvertised340AndReusesProbeForOrdinarySelection`, `blockstorage/volume_migration_test.go:TestMigrateVolumeRequiresAdvertisedSupportBeforeSelectedOrComputedVersion`.
- Unselected required calls reuse the same finite admitted advertisement to select ordinary cap3.71/maximum/optionalminimum. Missing/unsupported/malformed/IO/source/context gate evidence ends before POST with admitted discovery preserved.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestRevertVolumeToSnapshotRequiresAdvertised340AndReusesProbeForOrdinarySelection`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageFailuresKeepOnlyTheActualDiscoveryOrActionProof`.
- Package, service and Connection have required parameters and no options/builder. Context/provider/typed input preflight occurs before HTTP; Connection uses cached Cinder without CurrentLocation or Resource conversion.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`, `connectiontest/connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`, `connectiontest/connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanagePreflightAndGetterFaultsKeepCauses`.
- Action accepted response is opaque and independently owned; Completed acknowledges successful admission/read/close/source guard, while accepted faults retain partial Applied and joined causes without rollback/replay.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageFailuresKeepOnlyTheActualDiscoveryOrActionProof`.
- Original native HTTP rejection/retry policy and fixed physical request/source/live-auth boundaries cannot fabricate current Applied from discovery proof. No asynchronous wait, status/latest-snapshot validation or backend cleanup.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageFailuresKeepOnlyTheActualDiscoveryOrActionProof`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageLiveAuthAndBoundedRetryPreserveBodyAndOriginalHTTPPolicy`.

Python/Go differences:

- Source local snapshot Resource construction can expose arbitrary snapshot.id values; Go deliberately restricts this body operand to literal UTF-8 string without restricting it as a route. Detached/available/latest-snapshot documentation preconditions remain server concerns; no implementation GET/check is introduced.
- The actual required Source default comparison uses pinned Keystoneauth5.16.0 version_match: global latest is incompatible with major3, 3.latest can pass; actual Session canonicalization is a separately documented dependency boundary.
- Go safe UTF-8 current route ID and typed literal snapshot string replace mutable Resource/dict/Munch/untyped ID and constructor/descriptor/location/cache side effects. No lookup, model refresh or CurrentLocation is added.
- Source returns None and discards opaque action response; Go owns VolumeID/Microversion/Discovery/Applied/Completed. Completed means helper acknowledgement, not backend completion/removal/deletion.
- Finite call-owned discovery shares advertisement rather than reproducing Keystoneauth Adapter cache/link/envelope/exception order/Session canonicalization; selected literal forwarding remains a Go mapping. Empty selected Go string cannot separately represent Python explicit default empty versus None.
- Source status<400 maps finite Go100..399. Original native rejection remains native with no Applied; discarded rejected POST body IO is not claimed fully reconstructed. Fixed method/URL/body/framing/version and accepted phase proof are Go-owned policies.
- Only this actual pinned v3 Proxy declaration is classified. Separate native/v2/Resource/manage/read/metadata dependencies are not automatically promoted; Source static evidence and passed local Go QA do not prove authenticated Python/cloud runtime or full SDK completion.
- Current installed seven new groups directly verify literal/null bodies, required3.40 range/probe reuse, preflight/cache/location omission, native Unmanage separation, opaque/accepted Read+Close+source+customcontext proof and liveauth/bounded retry/body mutation. Ordinary unselected cap3.71 and global latest/3.latest version comparison use separately declared existing common-engine regression; no new operation-specific exhaustive redirect/framing matrix is claimed.

### `python:block_storage/v3/unmanage_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.UnmanageVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UnmanageVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.UnmanageVolume`

- UnmanageVolume sends exact {os-unmanage:null} to the safe current volume action route with no lookup, DELETE, options, force flag or model construction.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`, `connectiontest/connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`.
- No minimum gate is added. Selected nonempty version remains literal; otherwise finite ordinary discovery selects cap3.71/maximum/optionalminimum without source microversion mutation. Selected unmanage policy is tested directly; ordinary unselected cap details are existing shared Negotiate regression with actual same engine reuse.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`, `connectiontest/connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`, `blockstorage/volume_state_actions_test.go:TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap`.
- Package, service and Connection share required current volume ID and no caller builder/options; context/provider/input preflight and cached Cinder selection retain operation labels/causes without CurrentLocation.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`, `connectiontest/connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`, `connectiontest/connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanagePreflightAndGetterFaultsKeepCauses`.
- Opaque100..399 action response and independent Discovery/Applied proof expose helper Completed acknowledgement, not confirmed management removal or backend data deletion.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`, `connectiontest/connection_volume_revert_unmanage_test.go:TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody`.
- Accepted Read/Close/source/context failures preserve current Applied/all causes with Completedfalse and no SDK automatic replay/cleanup; original rejected HTTP remains native without borrowing discovery proof.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageFailuresKeepOnlyTheActualDiscoveryOrActionProof`, `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanageLiveAuthAndBoundedRetryPreserveBodyAndOriginalHTTPPolicy`.
- Existing native API.Unmanage is a separate unchanged {os-unmanage:{}}/202/error-only API; the new Source mapping exposes UnmanageVolume and does not broaden native/v2/Resource declarations.
  Tests: `blockstorage/volume_revert_unmanage_test.go:TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation`.

Python/Go differences:

- Native Gophercloud v2.15.0 Unmanage differs intentionally: empty object body, only202, ErrResult. The Source null body/new owned result is implemented independently rather than changing native semantics.
- Go safe UTF-8 current route ID replaces mutable Resource/dict/Munch/untyped ID and constructor/descriptor/location/cache side effects. No lookup, model refresh or CurrentLocation is added.
- Source returns None and discards opaque action response; Go owns VolumeID/Microversion/Discovery/Applied/Completed. Completed means helper acknowledgement, not backend completion/removal/deletion.
- Finite call-owned discovery shares advertisement rather than reproducing Keystoneauth Adapter cache/link/envelope/exception order/Session canonicalization; selected literal forwarding remains a Go mapping. Empty selected Go string cannot separately represent Python explicit default empty versus None.
- Source status<400 maps finite Go100..399. Original native rejection remains native with no Applied; discarded rejected POST body IO is not claimed fully reconstructed. Fixed method/URL/body/framing/version and accepted phase proof are Go-owned policies.
- Only this actual pinned v3 Proxy declaration is classified. Separate native/v2/Resource/manage/read/metadata dependencies are not automatically promoted; Source static evidence and passed local Go QA do not prove authenticated Python/cloud runtime or full SDK completion.
- Current installed seven new groups directly verify literal/null bodies, required3.40 range/probe reuse, preflight/cache/location omission, native Unmanage separation, opaque/accepted Read+Close+source+customcontext proof and liveauth/bounded retry/body mutation. Ordinary unselected cap3.71 and global latest/3.latest version comparison use separately declared existing common-engine regression; no new operation-specific exhaustive redirect/framing matrix is claimed.

### `python:block_storage/v3/attach_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeAttachInstance`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeAttachHostName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareCinderVolumeAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.AttachCinderVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeAttachInstance`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeAttachHostName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareCinderVolumeAttachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.AttachCinderVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AttachCinderVolume`

- Attach includes required literal mountpoint and selects Instance by nonnil presence, including empty; Instance wins HostName and only one selected field is sent. HostName fallback includes explicit empty. No mode, instance/host lookup, Nova request or state/poll is added.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentServiceMethodsPreserveNativeAttachDetachContracts`, `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`.
- Both selectors nil fail locally; only final selected selector UTF-8 is consumed, so inactive invalid HostName is ignored. Required mountpoint accepts empty/control/path-like UTF-8 text and fails invalid UTF-8 before direct originals/HTTP.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentDirectPreflightAndStickySourceStopOriginals`, `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentPreflightOptionsAndGetterFailuresKeepCauses`.
- Owned factories/full replacement and reusable HTTP-free Prepare copy nullable pointers, isolate returned policy from original inputs/results and reject missing/final-invalid selection with zero policy. No caller builder/interface is required.
  Tests: `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentOptionsFactoriesDefaultsAndReplacementOwnership`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`.
- Direct Source/current-ID/required-literal admission precedes original callbacks; originals snapshot their slice, execute once in order and copy retained policy before/after each. Source faults are sticky and stop later restoration callbacks with all context/callback causes.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentOriginalsOwnSlicePointersAndConnectorThroughHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentDirectPreflightAndStickySourceStopOriginals`.
- Connection preflight prepares originals once before required literal/current ID/cached Cinder getter; only Cinder is selected, configured CurrentLocation is not consumed, and retained values/getter/context/source faults preserve local operation/cause evidence.
  Tests: `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentPreflightOptionsAndGetterFailuresKeepCauses`.
- No action-specific minimum is added. Nonempty selected3.1/3.90/latest remains literal without discovery; unselected ordinary cap3.71/lower maximum/high minimum no-version policy owns separate admitted Discovery and action proof without source version mutation.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentUsesOrdinarySelectedOrCappedMicroversionWithoutMinimumGate`, `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`.
- Opaque accepted action bytes/Header/status are independently owned. Completed acknowledges admission/read/Close/source success; accepted Read/Close/source/custom-context failures retain Applied/independent ResponseError and all causes with Completedfalse and no SDK replay.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentAcceptedFailuresKeepIndependentActualProofAndJoinedCauses`.
- Live auth/reauth/bounded retry preserve original action body/selected version/admission and block changed body/header/source/physical307 body/framing/target before resend. Native Attach mode/both-selectors/202/error-only and nil service policy remain separate from SDK service AttachCinderVolume.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentServiceMethodsPreserveNativeAttachDetachContracts`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentNativeAuthRetriesAndPhysicalGuardsPreserveOriginalActionPolicy`.

Python/Go differences:

- Source nullable selectors can hold dynamic nonstring values; Go narrows to nullable UTF-8 strings and validates only selected instance/host. Explicit empty remains present. Direct Source attach has no mode; native Mode/omitempty/two nonempty selectors is preserved independently.
- Go safe current volume route ID and required UTF-8 body strings and nullable selectors replace supplied Resource/dict/Munch/None/untyped IDs, arbitrary dynamic annotations and constructor/descriptor/location/cache side effects; no name finder, refresh/model, CurrentLocation, wait/poll or additional server-state checks are introduced.
- Source action helpers return None after opaque response.status_code<400. Go finite100..399 policy adds owned VolumeID/Microversion/Discovery/Applied/Completed. Completed is helper acknowledgement, not server attachment/device state or backend readiness.
- Finite operation-owned ordinary discovery/max3.71 is an explicit Go mapping rather than complete Keystoneauth endpoint-data cache/link/envelope/exception ordering/Session header canonicalization. Nonempty selected literal is preserved; empty Go selection cannot separately represent Python explicit empty default versus None.
- Fixed method/route/serialized body/framing/selected version and admitted response ownership constrain retry/redirect while live native auth and allowed extra retry headers remain. Original native rejected HTTP is distinct and cannot become Applied; discarded rejected POST body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy attach_volume target is classified. Native/v2/Resource and existing cloud AttachVolume/DetachVolume workflows remain independently reviewed; the unchanged cloud boundary here is statically bound, not a new cloud workflow runtime test. No Python Source/dependency/authenticated OpenStack execution or full SDK completion is claimed.

### `python:block_storage/v3/detach_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeDetachForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithCinderVolumeDetachConnector`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareCinderVolumeDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.DetachCinderVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeDetachForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCinderVolumeDetachConnector`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareCinderVolumeDetachOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.DetachCinderVolume`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DetachCinderVolume`

- Detach Force nil/default/false sends os-detach with required attachment_id literal, including empty/control/path-like UTF-8. Inactive connector values/keys are not validated or sent, even if malformed. Force true selects os-force_detach without a separate lookup/wait/rollback workflow.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`, `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`.
- Force true omits nil/empty connector and includes all members of a nonempty object, including false/null/empty string and exact raw large/nested JSON values. Only active UTF-8 keys/complete UTF-8 JSON members are admitted; invalid/nil/trailing raw values fail locally and final repaired/inactive policy can succeed.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`.
- Owned false default pointer is fresh. Full replacement clears prior connector/force and restores defaultfalse; reusable WithOptions/Connector/Prepare own pointers/maps/every raw byte member, and failures return zero policy with joined context/callback causes.
  Tests: `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentOptionsFactoriesDefaultsAndReplacementOwnership`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses`.
- Original option slice/callback order/retained pointer-map-rawvalue ownership remain detached through HTTP. Direct Source/current ID/required literal preflight precedes callbacks; sticky source/cancel/callback errors stop later originals before physical proof.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentOriginalsOwnSlicePointersAndConnectorThroughHTTP`, `blockstorage/cinder_volume_attachment_options_test.go:TestCinderVolumeAttachmentDirectPreflightAndStickySourceStopOriginals`.
- Connection prepares originals once before body/route/cached Cinder admission; default/force calls need no Nova or CurrentLocation. Retained maps/getter mutation cannot alter the owned prepared request, while preflight/option/getter/context/source failures retain exact operation/cause without HTTP.
  Tests: `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`, `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached`, `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentPreflightOptionsAndGetterFailuresKeepCauses`.
- Normal and force detach use ordinary selected literal or finite cap3.71/max/optionalminimum negotiation with no minimum gate. Original client selection is not mutated and admitted Discovery/action proof are independent.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentUsesOrdinarySelectedOrCappedMicroversionWithoutMinimumGate`, `connectiontest/connection_cinder_volume_attachment_test.go:TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation`.
- Opaque acknowledgement and Completed remain distinct from backend disconnect/state completion. Accepted Read/Close/source/custom cancellation retain actual independent Applied/error proof and all causes without replay; rejected HTTP cannot borrow or fabricate admitted proof.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentAcceptedFailuresKeepIndependentActualProofAndJoinedCauses`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentNativeAuthRetriesAndPhysicalGuardsPreserveOriginalActionPolicy`.
- SDK service DetachCinderVolume normal/force/empty-ID/connector/result behavior remains distinct from native Detach empty omission/normal-only202/error-only. Live reauth/bounded retry/physical307 guards preserve original body/framing/URL/version/status policy and nil API preflight.
  Tests: `blockstorage/cinder_volume_attachment_test.go:TestCinderVolumeAttachmentServiceMethodsPreserveNativeAttachDetachContracts`, `blockstorage/cinder_volume_attachment_transport_test.go:TestCinderVolumeAttachmentNativeAuthRetriesAndPhysicalGuardsPreserveOriginalActionPolicy`.

Python/Go differences:

- Source arbitrary force truthiness and truthy connector objects are deliberately narrowed to nullable bool and owned JSON object map. Inactive connector is not consumed. Raw numeric precision/spelling/map serialization are Go JSON representation boundaries rather than arbitrary Python objects/insertion-order serialization.
- Source force action does not call backend disconnect or a separate rollback helper; Go force flag only chooses the action. Native Detach normal-only/omitempty202/error-only remains unchanged.
- Go safe current volume route ID and required UTF-8 body strings, nullable force bool and owned JSON connector replace supplied Resource/dict/Munch/None/untyped IDs, arbitrary dynamic annotations and constructor/descriptor/location/cache side effects; no name finder, refresh/model, CurrentLocation, wait/poll or additional server-state checks are introduced.
- Source action helpers return None after opaque response.status_code<400. Go finite100..399 policy adds owned VolumeID/Microversion/Discovery/Applied/Completed. Completed is helper acknowledgement, not server attachment/device state or backend readiness.
- Finite operation-owned ordinary discovery/max3.71 is an explicit Go mapping rather than complete Keystoneauth endpoint-data cache/link/envelope/exception ordering/Session header canonicalization. Nonempty selected literal is preserved; empty Go selection cannot separately represent Python explicit empty default versus None.
- Fixed method/route/serialized body/framing/selected version and admitted response ownership constrain retry/redirect while live native auth and allowed extra retry headers remain. Original native rejected HTTP is distinct and cannot become Applied; discarded rejected POST body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy detach_volume target is classified. Native/v2/Resource and existing cloud AttachVolume/DetachVolume workflows remain independently reviewed; the unchanged cloud boundary here is statically bound, not a new cloud workflow runtime test. No Python Source/dependency/authenticated OpenStack execution or full SDK completion is claimed.

### `python:block_storage/v3/set_volume_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataValue`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataRaw`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SetVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataValue`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataRaw`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SetVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SetVolumeImageMetadata`

- Set sends exactly one os-set_image_metadata action with metadata object, including omitted/nil/empty maps as {}. Literal empty/control/path-like strings and raw null/false/exact large/nested JSON members are preserved in the body; no lookup/model refresh/wait is added. Admitted opaque 200/203/204/302/399 responses complete acknowledgement without JSON parsing.
  Tests: `blockstorage/volume_image_metadata_test.go:TestSetVolumeImageMetadataPreservesEmptyMapsRawValuesAndOwnedFactories`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataServiceFacadePreservesNativeSetAndPublicPreflight`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`.
- String/raw/full factories replace policy; Value alone merges one string member. HTTP-free reusable Prepare owns maps/member bytes and fresh empty defaults, validates only final UTF-8 keys/strings/complete JSON values, permits later repair of invalid members, and returns zero policy on failures.
  Tests: `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOptionFactoriesOwnInputsReturnedValuesAndReplacementOrder`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareValidatesOnlyFinalMapsAndPresentKeys`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause`.
- Originals snapshot their callback slice, run once in order, copy intermediate maps/raw bytes and join callback/custom-context causes. Direct selected Source/current-ID admission precedes originals; sticky ResourceBase mutation stops a later restoration callback before HTTP.
  Tests: `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOriginalsOwnCallbackSliceIntermediateMapsAndKeySlices`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause`, `blockstorage/volume_image_metadata_test.go:TestSetVolumeImageMetadataPreservesEmptyMapsRawValuesAndOwnedFactories`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataServiceFacadePreservesNativeSetAndPublicPreflight`.
- Connection preflights context/provider, prepares originals once before current-ID/cached Cinder selection, ignores configured CurrentLocation, and detaches retained callback maps through the getter. Local/callback/getter/cached wrong-service failures preserve operation/cause and create no HTTP proof.
  Tests: `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataOriginalsRunOnceBeforeGetterAndRetainedOptionsStayDetached`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataPreflightOptionsAndGetterFailuresKeepEveryCause`.
- Nonempty selected 3.60/3.90 is literal and skips discovery; common ordinary action selection has no minimum gate and cap3.71. Malformed admitted unselected discovery blocks set with only its actual Discovery/ResponseError, while valid common batch negotiation is independently exercised by delete.
  Tests: `blockstorage/volume_image_metadata_test.go:TestSetVolumeImageMetadataPreservesEmptyMapsRawValuesAndOwnedFactories`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataNegotiatesOnceAndStopsAtCurrentNativeFailureWithoutBorrowedProof`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`.
- Accepted Read/Close/source/custom-context failures retain independently owned current Applied/error proof with Completedfalse and no SDK replay. Live auth/reauth/bounded retry keep original body/version/admission; retry/physical307 body/framing/URL/version changes are blocked and native rejection cannot fabricate Applied. Service native SetImageMetadata nil=>null/200-only/error-only remains independent.
  Tests: `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataAcceptedFailuresRetainOnlyCurrentPhaseProofAndJoinedCauses`, `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataNativeRetriesAndPhysicalMutationGuardsKeepBatchPolicy`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataServiceFacadePreservesNativeSetAndPublicPreflight`.

Python/Go differences:

- Python **metadata supplies a kwargs dictionary, including empty {}. Go string/raw/full factories replace request policy and Value merges; this is request assembly, not a claim that Cinder atomically replaces its stored image metadata.
- Go raw metadata permits owned ordinary JSON null/bool/numbers/containers without float preconversion, but rejects invalid UTF-8/incomplete JSON/nil raw members and excludes arbitrary Python objects/datetime/UUID/custom serializer behavior. Map member order and numeric/string/escape serialization are Go JSON representation boundaries, not Python mutable kwargs identity.
- Native API.SetImageMetadata remains independently map[string]string, nil metadata:null, HTTP200/error-only; ordinary MetadataIn childroutes are unchanged.
- Go safe explicit current volume ID and owned option/result values replace Source supplied Resource/dict/Munch/None/dynamic inherited ID and constructor/descriptor/location/session side effects. No name finder, CurrentLocation, refresh, state wait or model conversion is introduced.
- The Source helper returns None after response status<400. Go finite100..399 admission exposes owned actual stage responses and Completed acknowledgement; this does not prove the final server/backend metadata state.
- Go finite operation-owned ordinary discovery/cap3.71 and literal nonempty selected version differ from complete Keystoneauth endpoint-data cache/canonicalization/exception ordering. Empty Go selection cannot represent every distinct Python default None/empty value. Original client selection is not rewritten.
- Fixed method/URL/serialized-body/framing/version and captured Source constrain native retry/redirect while live provider authentication is allowed. Native original rejected HTTP remains native; discarded rejected HTTP body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy target is classified as go_mapping. Native/v2/Resource/MetadataMixin/ordinary MetadataIn and shared discovery dependencies are not promoted. Source/static comparison and Root offline Go HTTP/example checks do not imply Python/dependency/authenticated OpenStack execution or full SDK completion.

### `python:block_storage/v3/delete_volume_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataDeleteKeys`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataDeleteAll`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeImageMetadataDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.DeleteVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataDeleteKeys`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataDeleteAll`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeImageMetadataDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.DeleteVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteVolumeImageMetadata`

- Keys pointer nil/default means all; a nonnil pointer including nil/empty slice or zero-argument WithKeys means explicit none with zero HTTP/discovery after local admission. Explicit UTF-8 keys, including empty/control/path-like text, preserve order and duplicates as body literals with no GET; every final key is validated before earlier actions.
  Tests: `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataExplicitKeysKeepOrderDuplicatesAndEagerValidation`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOptionFactoriesOwnInputsReturnedValuesAndReplacementOrder`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareValidatesOnlyFinalMapsAndPresentKeys`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`.
- Default all intentionally corrects the pinned ordinary-metadata iteration bug: fixed-ID bodyless member GET admits sub400 but requires canonical volume object and reads only volume_image_metadata. Missing/null/empty image map emits no POST; nonobject/bad envelope/malformed JSON keeps actual Observed/error proof. Image keys are snapshotted/sorted, ignoring ordinary metadata, returned ID and unrelated typed fields.
  Tests: `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataAllReadsOnlyCanonicalImageFieldAndKeepsObservation`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`.
- Owned factories/full replacement/All/Keys and reusable HTTP-free Prepare preserve pointer presence, slice order/duplicates and returned-policy independence. Original callback slice and intermediate key backing arrays stay detached; final repaired keys succeed, and nil/callback/cancellation failures return zero policy with every cause.
  Tests: `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOptionFactoriesOwnInputsReturnedValuesAndReplacementOrder`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOriginalsOwnCallbackSliceIntermediateMapsAndKeySlices`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareValidatesOnlyFinalMapsAndPresentKeys`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataOriginalsRunOnceBeforeGetterAndRetainedOptionsStayDetached`.
- Direct Source/current-ID admission precedes original callbacks even for explicit none. Connection preflight prepares originals once before current-ID/cached Cinder selection, consumes no CurrentLocation, and validates unsafe IDs/invalid final keys/getter/cached source failures without member/action HTTP or borrowed proof.
  Tests: `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataExplicitKeysKeepOrderDuplicatesAndEagerValidation`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataServiceFacadePreservesNativeSetAndPublicPreflight`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataOriginalsRunOnceBeforeGetterAndRetainedOptionsStayDetached`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataPreflightOptionsAndGetterFailuresKeepEveryCause`.
- A nonempty selected version skips discovery; otherwise one ordinary cap3.71 negotiation supplies the entire batch GET/POST version and separate actual Discovery. First native action failure stops later keys, retains earlier Deleted/optional Observed, records Failed key with nil Response while the returned error retains current native status/body/header, and cannot borrow admitted ResponseError proof.
  Tests: `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataNegotiatesOnceAndStopsAtCurrentNativeFailureWithoutBorrowedProof`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`, `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataNativeRetriesAndPhysicalMutationGuardsKeepBatchPolicy`.
- Per-key opaque acknowledgement bodies/headers and member/error proofs own independent bytes. Accepted IO/source/custom cancellation retains only the current Observed or Failed.Response plus earlier completed keys, without replay/rollback; Completed only means helper acknowledgement. Live auth and bounded retry preserve batch policy, physical bodyless GET/POST/URL/version/framing mutations are blocked, and rejected injected GET Body Close runs once with close cause retained.
  Tests: `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataExplicitKeysKeepOrderDuplicatesAndEagerValidation`, `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataAllReadsOnlyCanonicalImageFieldAndKeepsObservation`, `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataAcceptedFailuresRetainOnlyCurrentPhaseProofAndJoinedCauses`, `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataNativeRetriesAndPhysicalMutationGuardsKeepBatchPolicy`.

Python/Go differences:

- Pinned keys=None calls Volume.delete_image_metadata, which loops inherited ordinary self.metadata rather than volume_image_metadata. ID-only missing ordinary metadata defaults to None and implies iteration TypeError by static inspection; no Python execution is asserted. Go intentionally repairs this branch with fresh fixed-ID image-field GET and sorted snapshot keys.
- The repaired all path is not an atomic clear: keys added after observation are not included, concurrent changes can cause a later failure, earlier deletions remain, and there is no implicit merge/write/retry/rollback promise.
- Python non-None keys can be an arbitrary lazy iterable, string character iteration or generator failing after earlier POSTs. Go narrows to an eager owned UTF-8 string slice; nonnil pointer to nil/empty slice explicitly means none. Final key validation occurs before any mutation and duplicate order is preserved.
- Source selects microversion separately for every key action using mutable session state. Go negotiates/captures one version for the whole batch, including the added default-all GET; this deliberate stable policy does not reproduce per-key session mutation.
- Observed/Discovery/Deleted/Failed and current native/error proof are distinct owned phases. A failure before attempting the next key can leave Failed nil, while an attempted native rejected action has Failed with nil Response. Earlier acknowledgement is not borrowed as current evidence.
- Go safe explicit current volume ID and owned option/result values replace Source supplied Resource/dict/Munch/None/dynamic inherited ID and constructor/descriptor/location/session side effects. No name finder, CurrentLocation, refresh, state wait or model conversion is introduced.
- The Source helper returns None after response status<400. Go finite100..399 admission exposes owned actual stage responses and Completed acknowledgement; this does not prove the final server/backend metadata state.
- Go finite operation-owned ordinary discovery/cap3.71 and literal nonempty selected version differ from complete Keystoneauth endpoint-data cache/canonicalization/exception ordering. Empty Go selection cannot represent every distinct Python default None/empty value. Original client selection is not rewritten.
- Fixed method/URL/serialized-body/framing/version and captured Source constrain native retry/redirect while live provider authentication is allowed. Native original rejected HTTP remains native; discarded rejected HTTP body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy target is classified as go_mapping. Native/v2/Resource/MetadataMixin/ordinary MetadataIn and shared discovery dependencies are not promoted. Source/static comparison and Root offline Go HTTP/example checks do not imply Python/dependency/authenticated OpenStack execution or full SDK completion.

### `python:block_storage/v3/upload_volume_to_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadDiskFormat`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadContainerFormat`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadVisibility`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageUploadProtected`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeImageUploadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.UploadVolumeToImage`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadDiskFormat`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadContainerFormat`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadVisibility`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageUploadProtected`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeImageUploadOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.UploadVolumeToImage`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UploadVolumeToImage`

- The required image name is literal UTF-8 text, including empty/control/path-like strings. Force defaults to an owned false pointer and is always sent; nil formats/visibility/protected are omitted, while explicit empty formats/visibility and false protected are included without format/visibility enums or invented defaults.
  Tests: `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageSerializesLiteralNamesDefaultsAndOptionalPresence`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageDirectPreflightAndOwnedOriginalsKeepFixedRequestPolicy`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadOptionFactoriesOwnPointersDefaultsAndFullReplacement`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadPrepareValidatesOnlyFinalUTF8AndPreservesLiteralStrings`.
- Only nonnil Visibility or Protected requests advertised 3.1 support, including empty/false. Both server bounds must contain fixed3.1 and any selected version must have the same major and satisfy3.1; a selected version above the advertised maximum can still pass. No-presence selected3.0 skips discovery; unselected required selection shares one probe and uses the ordinary3.71 cap/lower server maximum without rewriting the client.
  Tests: `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageConditional31GateUsesAdvertisedBoundsAndOrdinaryActionSelection`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageSerializesLiteralNamesDefaultsAndOptionalPresence`, `connectiontest/connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadUsesCachedCinderAndGatesOnlyOptionalPresence`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageServiceKeepsNativeUploadImageOmission202AndTypedResult`.
- Owned factories/full replacement and HTTP-free Prepare preserve all pointer presence and reusable policy independence. The original option slice is copied before callbacks; every intermediate/final pointer is detached, final UTF-8 can be repaired by later replacement, and nil/callback/custom-cancellation failures return zero policy with every cause. Direct Source/ID/image admission precedes originals; Connection prepares originals once before image/ID/cached Cinder selection and consumes no CurrentLocation.
  Tests: `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadOptionFactoriesOwnPointersDefaultsAndFullReplacement`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadOriginalsOwnCallbackSliceAndIntermediatePointers`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadPrepareValidatesOnlyFinalUTF8AndPreservesLiteralStrings`, `blockstorage/volume_image_upload_options_test.go:TestVolumeImageUploadPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageDirectPreflightAndOwnedOriginalsKeepFixedRequestPolicy`, `connectiontest/connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadOriginalsRunOnceBeforeGetterAndRetainedPointersStayDetached`, `connectiontest/connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadPreflightOptionsLiteralAndGetterFailuresKeepEveryCause`, `connectiontest/connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadUsesCachedCinderAndGatesOnlyOptionalPresence`.
- Accepted action handling requires one UTF-8 JSON object and the exact os-volume_upload_image key. Its selected value can be any JSON shape including null, nonobject and object without image_id; interior whitespace/numeric/escape literals are preserved and the last outer duplicate wins. Missing/bad key/root/malformed/trailing/UTF8/empty204 failures retain current Applied and independent error proof, keep Upload nil/Completed false and issue no parser replay.
  Tests: `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageAcceptsArbitrarySelectedJSONAndKeepsFailedDecodeProof`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageServiceKeepsNativeUploadImageOmission202AndTypedResult`, `connectiontest/connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadUsesCachedCinderAndGatesOnlyOptionalPresence`.
- Discovery, Applied, selected Upload and response-error snapshots own independent bytes/headers. Accepted Read/Close/source/custom-cancellation failures keep the current actual phase and every cause, retain earlier completed discovery independently, close once and leave Upload nil/Completed false. Completed reports successful helper response handling without a Glance lookup, image wait, refresh or rollback.
  Tests: `blockstorage/volume_image_upload_transport_test.go:TestUploadVolumeToImageAcceptedDiscoveryOrActionFailureKeepsCurrentProofAndNoValue`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageAcceptsArbitrarySelectedJSONAndKeepsFailedDecodeProof`, `connectiontest/connection_volume_image_upload_test.go:TestConnectionVolumeImageUploadUsesCachedCinderAndGatesOnlyOptionalPresence`.
- Live native401 reauthentication and bounded503 retry preserve the original serialized upload body, captured ordinary headers, chosen version and100..399 admission. Retry body/version/source mutations and physical redirected body/header/target/framing mutations block resend; original rejected HTTP retains native status/body/header with Applied nil and no borrowed discovery proof. The service facade keeps native UploadImage omitempty/202-only/typed VolumeImage behavior independently unchanged.
  Tests: `blockstorage/volume_image_upload_transport_test.go:TestUploadVolumeToImageLiveNativeRetriesAndPhysicalGuardsKeepBodyAndOriginalAdmission`, `blockstorage/volume_image_upload_test.go:TestUploadVolumeToImageServiceKeepsNativeUploadImageOmission202AndTypedResult`.

Python/Go differences:

- Go safe explicit current volume ID and UTF-8 literal image name replace Source supplied Resource/dict/Munch/None/dynamic inherited ID and constructor/descriptor/location/cache/session side effects. This helper adds no name finder, CurrentLocation, Volume model refresh, Glance lookup, image-byte upload, wait or rollback.
- Source annotations do not constrain runtime arguments: its own extended Proxy mock test forwards protected="5", and Python None/dynamic force can be forwarded literally. Go uses nullable typed strings/bools; Force nil means defaultfalse, optional nil means omission, and explicit empty/false remains present. Arbitrary Python runtime objects/custom serializers are outside this typed mapping.
- Source performs the3.1 requirement through endpoint-data utilities, then ordinary action selection through the Resource. Go captures one Source and uses finite operation-owned discovery/shared3.71 selection; this does not reproduce full Keystoneauth endpoint cache/getter count/canonicalization/exception order. Nonempty selected text is preserved; Go empty selection cannot represent every distinction between Python None and explicit empty default.
- The fixed3.1 gate compares3.1 with both advertised bounds and selected same-major minimum, not selected against the server maximum. A selected3.90 can pass advertised3.0..3.50 and is sent literally; global latest fails the same-major gate whereas3.latest passes. Dependency Session normalization/header behavior remains a separate declared boundary.
- The Source cast(dict, ...) is only an annotation cast and returns the exact selected JSON value; image_id and object shape are not required. Go returns owned literal RawMessage, preserving interior numeric/escape/whitespace spelling rather than Python mutable parsed objects/float rounding. Outer duplicate envelope keys follow ordinary last-key JSON decoding; successful null is nonnil literalnull, unlike error nil Upload.
- Go validates accepted UTF-8 JSON and an exact object envelope before exposing Upload, and exposes separate Discovery/Applied/Upload/Completed evidence. Accepted decode/IO/source/context faults keep actual completed HTTP proof without replay. These error/evidence choices do not promise identical Python exception taxonomy or final image/backend availability.
- Source raise_from_response admits status<400; Go uses finite100..399 and constrains native retry/redirect through fixed physical method/URL/body/framing/version and captured Source while permitting live provider auth. Original rejected HTTP remains native; discarded native rejected-body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy upload_volume_to_image target is classified as go_mapping. Native UploadImage omitempty/202-only/typed VolumeImage, Glance image.Service.UploadImage, v2/Resource/shared discovery dependencies are not promoted. Static Source/dependency comparison and Root offline Go HTTP/example checks do not imply Python/dependency/authenticated OpenStack runtime or full SDK completion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func mergeVolumeMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]string, error) {
    scope, err := volumes.New(client).MetadataIn(ctx, resource.ID(id))
    if err != nil {
        return nil, err
    }
    result, err := scope.Merge(ctx, map[string]string{"owner": "team-a"})
    if err != nil {
        return nil, err
    }
    return result.Metadata, nil // full actual server response, including retained keys
}
```

## References

- [shared metadata guide](../../metadata/README.md)
- [v2 guide](../../v2/volumes/README.md)
- [HTTP contract tests](../../../api/cinder_metadata_scopes_test.go)
- [Connection tests](../../../connectiontest/connection_metadata_test.go)
- [release-pinned server contract](../../../docs/cinder-metadata-server-contracts.md)
- [Service Waits](../../../docs/service-waits.md)
- [the package and Connection examples](../../volume-actions.md)
- [the Connection and package flag examples](../../volume-flags.md)
- [the Connection and package examples](../../volume-resize-retype.md)
- [Volume Migration Reset](../../volume-migration-reset.md)
- [Volume Revert Unmanage](../../volume-revert-unmanage.md)
- [Cinder Volume Attachment](../../cinder-volume-attachment.md)
- [Volume Image Metadata](../../volume-image-metadata.md)
- [Volume Upload Image](../../volume-upload-image.md)
