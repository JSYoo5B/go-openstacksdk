# Volumes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v2/delete_volume_metadata`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.DeleteKeys`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithMetadataHeaders`

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

Python/Go differences:

- Go uses explicit resource.Ref and a fixed scope returning owned actual HTTP evidence; Python accepts a string ID or existing Resource and returns/mutates that seeded Resource. Explicit Name lookup is a Go convenience and retains the existing native resolver boundary.
- Metadata values are concrete Go strings. Exact lowercase metadata objects and string members are required; Python metadata uses a dict descriptor and its own json/update behavior. Unknown response envelope fields and their original number spelling remain in raw Body, not in a widened typed metadata map.
- Go closes per-call options to SDK-owned headers, preserves original source ownership and exposes context/error explicitly. It does not reproduce injected Python sessions, custom Resource URI/adapter selection or automatic Resource microversion negotiation.
- No new metadata-specific microversion gate is introduced. Fixed Cinder 27.0.0 proves v3 routes and Volume GET/conditional collection PUT ETag at 3.15+, not current legacy v2 deployment support or an equivalent Snapshot ETag policy; caller/server own that condition.
- Literal ID/key components are escaped once and invalid route inputs are rejected. Mock routing evidence is not a guarantee of encoded slash/dot handling through a real proxy/WSGI or of server key/length policy acceptance.
- Python updates the Resource metadata cache after each acknowledged DELETE and can raise from a stale/missing local key; Go records actual partial acknowledgements without a cache mutation. Non-HTTP empty-key results contain no synthesized success evidence, and 404 is not ignored.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/fetch_volume_metadata`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithMetadataHeaders`

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

Python/Go differences:

- Go uses explicit resource.Ref and a fixed scope returning owned actual HTTP evidence; Python accepts a string ID or existing Resource and returns/mutates that seeded Resource. Explicit Name lookup is a Go convenience and retains the existing native resolver boundary.
- Metadata values are concrete Go strings. Exact lowercase metadata objects and string members are required; Python metadata uses a dict descriptor and its own json/update behavior. Unknown response envelope fields and their original number spelling remain in raw Body, not in a widened typed metadata map.
- Go closes per-call options to SDK-owned headers, preserves original source ownership and exposes context/error explicitly. It does not reproduce injected Python sessions, custom Resource URI/adapter selection or automatic Resource microversion negotiation.
- No new metadata-specific microversion gate is introduced. Fixed Cinder 27.0.0 proves v3 routes and Volume GET/conditional collection PUT ETag at 3.15+, not current legacy v2 deployment support or an equivalent Snapshot ETag policy; caller/server own that condition.
- Literal ID/key components are escaped once and invalid route inputs are rejected. Mock routing evidence is not a guarantee of encoded slash/dot handling through a real proxy/WSGI or of server key/length policy acceptance.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_volume_metadata`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithMetadataHeaders`

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

Python/Go differences:

- Go uses explicit resource.Ref and a fixed scope returning owned actual HTTP evidence; Python accepts a string ID or existing Resource and returns/mutates that seeded Resource. Explicit Name lookup is a Go convenience and retains the existing native resolver boundary.
- Metadata values are concrete Go strings. Exact lowercase metadata objects and string members are required; Python metadata uses a dict descriptor and its own json/update behavior. Unknown response envelope fields and their original number spelling remain in raw Body, not in a widened typed metadata map.
- Go closes per-call options to SDK-owned headers, preserves original source ownership and exposes context/error explicitly. It does not reproduce injected Python sessions, custom Resource URI/adapter selection or automatic Resource microversion negotiation.
- No new metadata-specific microversion gate is introduced. Fixed Cinder 27.0.0 proves v3 routes and Volume GET/conditional collection PUT ETag at 3.15+, not current legacy v2 deployment support or an equivalent Snapshot ETag policy; caller/server own that condition.
- Literal ID/key components are escaped once and invalid route inputs are rejected. Mock routing evidence is not a guarantee of encoded slash/dot handling through a real proxy/WSGI or of server key/length policy acceptance.
- The pinned get_volume_metadata method is a deprecated alias warning about SDK 5.0 removal and delegating to fetch_metadata. Go reuses MetadataScope.Get and does not add a duplicate deprecated warning-emitting method.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/set_volume_metadata`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.MetadataIn`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.Merge`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.Replace`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.ID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.MetadataScope.RawClient`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithMetadataHeaders`

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

Python/Go differences:

- Go uses explicit resource.Ref and a fixed scope returning owned actual HTTP evidence; Python accepts a string ID or existing Resource and returns/mutates that seeded Resource. Explicit Name lookup is a Go convenience and retains the existing native resolver boundary.
- Metadata values are concrete Go strings. Exact lowercase metadata objects and string members are required; Python metadata uses a dict descriptor and its own json/update behavior. Unknown response envelope fields and their original number spelling remain in raw Body, not in a widened typed metadata map.
- Go closes per-call options to SDK-owned headers, preserves original source ownership and exposes context/error explicitly. It does not reproduce injected Python sessions, custom Resource URI/adapter selection or automatic Resource microversion negotiation.
- No new metadata-specific microversion gate is introduced. Fixed Cinder 27.0.0 proves v3 routes and Volume GET/conditional collection PUT ETag at 3.15+, not current legacy v2 deployment support or an equivalent Snapshot ETag policy; caller/server own that condition.
- Literal ID/key components are escaped once and invalid route inputs are rejected. Mock routing evidence is not a guarantee of encoded slash/dot handling through a real proxy/WSGI or of server key/length policy acceptance.
- Python set_metadata caches the request delta and returns self rather than translating the complete merged response. Go Merge and explicit Replace return server values, have no cached lifecycle, and send metadata:{} for nil/empty inputs; native Snapshot omitempty behavior remains separate.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/wait_for_status`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForAvailable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.WaitForAvailable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.WaitForAvailable`

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

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/wait_for_delete`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.WaitForDelete`

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

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes"
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
- [v3 guide](../../v3/volumes/README.md)
- [HTTP contract tests](../../../api/cinder_metadata_scopes_test.go)
- [Connection tests](../../../connection_metadata_test.go)
- [release-pinned server contract](../../../docs/cinder-metadata-server-contracts.md)
- [Service Waits](../../../docs/service-waits.md)
