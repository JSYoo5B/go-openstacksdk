# Metadata Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
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
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
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
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
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
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
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
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
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
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
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
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
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
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesRoutesAndOwnedResponse`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks`, `internal/cindermetadata/scope_test.go:TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP`.
- Explicit resource.Name is a Go convenience resolving once through the current parent collection, then validating and fixing the resolved ID. Missing/duplicate/error observations retain the existing resolver policy: v2 Snapshot uses its native single page, while the other three bindings use native linked continuation; this scope does not broaden those native pagers.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `connection_metadata_test.go:TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences`.
- Input metadata maps and deletion keys are copied before custom options and HTTP; concrete WithMetadataHeader/WithMetadataHeaders capture header values and allow sequential canonical last-wins reuse without an application builder. Returned fields do not serve as request inputs or retarget identity.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesNameBindingAndRequestSnapshots`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation`, `blockstorage/metadata/options_test.go:TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse`, `blockstorage/metadata/options_test.go:TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap`.
- Malformed accepted JSON, missing/null/nonobject exact metadata, non-string/null values and accepted partial reads retain raw HTTP evidence through ResponseError and original causes. Native unexpected status, transport and cancellation remain unwrap-able; accepted decode failure does not reenter the SDK request or invoke retry callbacks. Invalid UTF-8 accepted response bytes are also terminal and retain their original evidence.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors`, `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `internal/cindermetadata/scope_test.go:TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse`, `internal/cindermetadata/scope_test.go:TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal`.
- Closed metadata options accept headers only: auth/body/routing/version/field/query/argument injection and ambiguous case conflicts fail before HTTP. Configured ordinary MoreHeaders retains native precedence. A differing configured versus explicit If-Match is rejected, while ETag/If-Match/412 are preserved without automatic GET, version upgrade or retry after 412.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesHeaderVersionAndETagPolicies`, `internal/cindermetadata/scope_test.go:TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities`.
- The fixed URL keeps the selected ResourceBase/reverse prefix and original live Provider, middleware, token, configured microversion and caller context. Existing Provider reauthentication and explicitly configured native retry remain in use; the scope introduces no automatic server concurrency policy.
  Tests: `api/cinder_metadata_scopes_test.go:TestCinderMetadataScopesFixedTargetLiveProviderAndContext`, `connection_metadata_test.go:TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix`, `internal/cindermetadata/scope_test.go:TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget`.
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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--volume", default="")
parser.add_argument("--snapshot", default="")
parser.add_argument("--action", choices=["fetch", "get", "set", "delete", "clear", "noop"], default="fetch")
parser.add_argument("--metadata", default='{"sdk-example":"python"}')
parser.add_argument("--key", default="sdk-example")
args = parser.parse_args()
if not args.volume and not args.snapshot:
    parser.error("provide --volume or --snapshot")

conn = openstack.connect()
for kind, identity in [("volume", args.volume), ("snapshot", args.snapshot)]:
    if not identity:
        continue
    proxy = conn.block_storage
    if args.action in ("fetch", "get"):
        record = getattr(proxy, f"{args.action}_{kind}_metadata")(identity)
        print(kind, "metadata", record.metadata)
    elif args.action == "set":
        record = getattr(proxy, f"set_{kind}_metadata")(identity, **json.loads(args.metadata))
        print(kind, "cached request delta", record.metadata)
    else:
        keys = None if args.action == "clear" else ([] if args.action == "noop" else [args.key])
        result = getattr(proxy, f"delete_{kind}_metadata")(identity, keys=keys)
        print(kind, "delete return", result)  # Source returns None.
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func show(kind string, declared *resource.RawResource, envelope json.RawMessage, status int) {
    metadata := "<no declared view>"
    if declared != nil { metadata = string(declared.Body["metadata"]) }
    fmt.Printf("%s metadata=%s status=%d actual_body=%q\n", kind, metadata, status, envelope)
}

func main() {
    volumeID := flag.String("volume", "", "existing volume ID")
    snapshotID := flag.String("snapshot", "", "existing snapshot ID")
    action := flag.String("action", "fetch", "fetch, get, set, delete, clear, or noop")
    metadataJSON := flag.String("metadata", `{"sdk-example":"go"}`, "metadata JSON object")
    key := flag.String("key", "sdk-example", "one metadata key to delete")
    flag.Parse()
    if *volumeID == "" && *snapshotID == "" { log.Fatal("provide -volume or -snapshot") }
    switch *action { case "fetch", "get", "set", "delete", "clear", "noop": default: log.Fatal("unknown action") }
    // RawMessage values preserve JSON null/false/zero and large integers.
    var members map[string]json.RawMessage
    if err := json.Unmarshal([]byte(*metadataJSON), &members); err != nil { log.Fatal(err) }
    values := make(map[string]any, len(members))
    for name, raw := range members { values[name] = raw }
    var keys []string
    if *action == "noop" { keys = []string{} }
    if *action == "delete" { keys = []string{*key} }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }

    if *volumeID != "" {
        var record *volumes.VolumeResourceRecord
        switch *action {
        case "fetch": record, err = service.Volumes.FetchMetadataRecord(ctx, *volumeID)
        case "get": record, err = service.Volumes.GetMetadataRecord(ctx, *volumeID)
        case "set": record, err = service.Volumes.SetMetadataRecord(ctx, *volumeID, values,
            volumes.WithRecordMetadataHeader("X-Client-Request-Id", "metadata-volume"))
        default:
            var deleted *volumes.MetadataRecordDeletion
            deleted, err = service.Volumes.DeleteMetadataRecord(ctx, *volumeID, keys)
            if deleted != nil {
                record = deleted.Record
                fmt.Printf("volume clear=%t completed_cached_keys=%v accepted_receipts=%d\n", deleted.Cleared, deleted.DeletedKeys, len(deleted.Responses))
            }
        }
        if record != nil { show("volume", record.Resource, record.Envelope, record.StatusCode) }
        if err != nil { log.Fatal(err) }
    }
    if *snapshotID != "" {
        var record *snapshots.SnapshotRecord
        switch *action {
        case "fetch": record, err = service.Snapshots.FetchMetadataRecord(ctx, *snapshotID)
        case "get": record, err = service.Snapshots.GetMetadataRecord(ctx, *snapshotID)
        case "set": record, err = service.Snapshots.SetMetadataRecord(ctx, *snapshotID, values,
            snapshots.WithRecordMetadataHeader("X-Client-Request-Id", "metadata-snapshot"))
        default:
            var deleted *snapshots.MetadataRecordDeletion
            deleted, err = service.Snapshots.DeleteMetadataRecord(ctx, *snapshotID, keys)
            if deleted != nil {
                record = deleted.Record
                fmt.Printf("snapshot clear=%t completed_cached_keys=%v accepted_receipts=%d\n", deleted.Cleared, deleted.DeletedKeys, len(deleted.Responses))
            }
        }
        if record != nil { show("snapshot", record.Resource, record.Envelope, record.StatusCode) }
        if err != nil { log.Fatal(err) }
    }
}
```

## References

- [Cinder API](https://docs.openstack.org/api-ref/block-storage/v3/)
- [Policy](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)
- [Python Metadata Test](volumes/python_metadata_test.go)
- [Python Metadata Test](snapshots/python_metadata_test.go)
