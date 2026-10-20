# Senlin Server Contracts

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/create_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Create`

- POST201 published and POST202 controller creation preserve omitted capacity/size/timeout defaults, explicit zero/-1/null and raw config/metadata, require name/profile_id, and preserve an optional201 or mandatory202 action Location.
  Tests: `api/clustering_clusters_test.go:TestClusteringClustersCreateGetExactModelAndSubmission`, `api/clustering_clusters_test.go:TestClusteringClustersCreateDefaultsOmitOptionalFields`, `api/clustering_clusters_response_test.go:TestClusteringClustersAcceptedMalformedObjectRetainsEvidenceWithoutResend`, `api/clustering_clusters_test.go:TestClusteringClustersMutationSchemasProtectCoreAndResponseFields`, `api/clustering_async_location_case_test.go:TestClusteringClusterMutationLocationHeaderCaseAndAmbiguity`, `connection_senlin_async_test.go:TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes`, `api/clustering_scalar_options_test.go:TestClusteringScalarOptionsPreserveExplicitValuesAndReuse`.
- The verified16.0.0 router accepts creation with202; both201 and202 preserve the cluster model and response evidence, while a202 without one valid selected-service action Location fails without a second POST. Undeclared200 and malformed accepted bodies retain their distinct HTTP/decode causes.
  Tests: `api/clustering_cluster_create_status_test.go:TestClusteringClusterCreatePublishedAndControllerSuccessCodes`, `api/clustering_cluster_create_status_test.go:TestClusteringClusterCreateAcceptedRequiresValidActionLocation`, `api/clustering_cluster_create_status_test.go:TestClusteringClusterCreateUndeclaredStatusAndMalformedAcceptedBody`.

Python/Go differences:

- Go uses concrete owned options, context and explicit ID/Name references rather than mutable Python Resource input; With scalar/null helpers preserve omission, explicit zero/false/empty and null without requiring caller builders.
- Documented singular envelopes and success codes are strict; raw HTTP/header/body evidence and exact JSON numbers are retained instead of Python permissive fallback decoding or cached Resource mutation.
- The selected numeric microversion and protected field/query/header extensions are explicit; null field support remains deployment-defined and no unverified force version gate is invented.
- Accepted Location must identify one action at the selected origin and collection, including reverse-proxy prefixes; missing required, empty, ambiguous, foreign or malformed references return ResponseError without following or resending. Submission holds action identity and original response, not an invented action status or completed result.
- Go accepts published201 and verified16.0.0 controller202;202 is an asynchronous submission and requires action evidence. This release-source audit and local HTTP test do not claim live-cloud acceptance.

### `python:clustering/v1/get_cluster_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.API.InCluster`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.Scope.Get`

- GET200 uses the selected fixed canonical cluster and a controller policy name, short ID or UUID directly; the binding ID remains distinct from policy_id and no separate policy lookup or incidental Location fetch is performed.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesCanonicalScopeAndDirectPolicyRoutes`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactBodyIdentityOwnsRoutes`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactNameIgnoresCaseFoldedExtensions`.
- The documented cluster_policy object preserves raw numbers, unknown/null fields, copied HTTP metadata and canonical lowercase identity fields; missing or invalid IDs and a foreign parent produce whole accepted-response evidence, while unexpected codes and canceled contexts retain their original causes without a second binding request.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesRequiredRowsRetainWholeResponse`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesWrongCodesRemainNative`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesParentIdentityEvidenceAndSourceRecheck`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesPreflightAndScopedWaits`.

Python/Go differences:

- Go uses context, explicit ID/Name parent references and independent response snapshots instead of Python proxy._get_resource reusing and updating a mutable ClusterPolicy instance. This declared Get mapping does not claim inherited Resource fetch/cache controls.
- InCluster adds a parent GET or exact Name-list search, requires its canonical lowercase raw id and fixes that parent once. Python _proxy.py:1097-1099 passes cluster directly as the URI attribute without that confirmation request.
- Go requires separate valid raw id, policy_id and matching cluster_id fields and restores exact lowercase identity/name values despite case-folded JSON extensions. Python Resource.id can fall back to the alternate policy_id and its Body/URI descriptors normalize resource attributes.
- Go accepts only HTTP200 and the singular object envelope, retaining accepted-response errors and raw HTTP evidence instead of Python permissive flat/empty fallback and cache mutation. URIClusterID is separate from the response Body; Location is not interpreted as an action.
- Source authentication, endpoint prefix and selected Senlin version remain explicit and are checked before scope creation and binding reads; this base operation has no invented minimum microversion.

### `python:clustering/v1/cluster_policies`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.API.InCluster`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.Scope.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.Scope.All`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.WithListHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.WithListMicroversion`

- Lazy GET200 list queries map is_enabled to enabled, preserve explicit false and policy_name/policy_type/sort plus owned vendor-query snapshots, repeat independently and stop after consumer break. Only advertised next links supply a marker; response binding or policy IDs never invent a marker fallback.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesLazyLinkedPagesSnapshotsAndNoFallback`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesPreflightAndScopedWaits`.
- Each row retains independent raw JSON/HTTP metadata and separate canonical binding/policy/parent identities. Foreign continuation parents, malformed rows and source changes stop the collection with original accepted-page or native HTTP evidence.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesRequiredRowsRetainWholeResponse`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactBodyIdentityOwnsRoutes`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExplicitLinksKeepParentAndSource`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactNameIgnoresCaseFoldedExtensions`.
- Binding MaxItems is a local raw-row cap without a wire-limit hint; Paginated=False consumes the first response. Snapshot bool pointers, later-option precedence and zero-unlimited defaults remain independent across lazy repeated iterations. Raw control query keys and initial limit/marker fail before list HTTP, while cap/empty stops ignore unused invalid rows/links and uncapped row errors retain whole-page evidence.
  Tests: `api/clustering_catalog_list_controls_test.go:TestClusteringBindingListControlsStayLocalAndOwnSnapshots`.
- Public Resources.List/All forwards raw-row caps and first-response controls into the service iterator before applicable local name/status filters, retains per-row HTTP evidence and selected origin guards, and honors the binding limit-hint policy. Empty pages terminate even with invalid next links. Binding caps supply no wire-limit hint and initial page size remains unsupported.
  Tests: `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsSinglePage`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize`, `api/clustering_resource_list_controls_test.go:TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard`.
- Known pinned Body fields and Python attribute aliases can be selected explicitly for local raw JSON comparison, including query-capable fields; unknown/source-excluded fields and raw Body query extensions fail before child HTTP.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSourceAliasesAndQueryCapableFieldsStayLocal`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersKnownQueryUnknownFieldsAndNamespacesFailBeforeChildHTTP`.
- Filter JSON and canonical maps are owned snapshots, alias/canonical options use last-wins and each iterator traversal owns its prepared arguments; missing and explicit null compare as null without erasing raw presence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersSixFacadesSnapshotAliasesAndReuse`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersMissingAndExplicitNullRemainDistinctFromValues`, `internal/senlin/filter_options_test.go:TestBodyFilterOptionsSnapshotCanonicalAliasesAndOwnRawMaps`.
- Raw pagination and validation precede filtering: nonmatching rows consume caps and determine markers, single-page/cap stops skip unused continuation.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCapsAndSinglePageBeforeMatching`, `api/clustering_body_filters_test.go:TestClusteringBodyFiltersFilteredPagesKeepRawMarkersAndParent`.
- Nested object subsets, exact large decimal values, arrays of complete objects, bool/number distinction and empty-object behavior are verified using the real typed facade.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersNestedSubsetsExactNumbersArraysAndBooleans`.
- Raw type id filters do not synthesize a name fallback, while the binding scope still requires a valid raw binding ID before local filtering and preserves failure evidence.
  Tests: `api/clustering_body_filters_test.go:TestClusteringBodyFiltersRawCatalogIDDoesNotFallbackAndBindingValidationPrecedesFilter`.
- Typed List/All owns explicit per-call header and numeric microversion selection without changing the source client. Canonical case-insensitive caller headers override source defaults; reusable option slices and config maps are snapshotted for independent traversals and concurrent sibling calls.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestElevenFacadesWireAndSourceOwnership`, `api/clustering_list_request_test.go:TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot`, `api/clustering_list_request_test.go:TestClusteringListRequestIteratorsReuseAndConcurrentSiblings`, `internal/senlin/list_request_test.go:TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration`.
- All pages retain the selected per-call headers/version while authenticated requests use the shared provider live token, including reauthentication replay. Per-page source validation and accepted/native error evidence remain intact; unused rows/links still stop at the existing local cap or first-page boundary.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken`, `api/clustering_list_request_test.go:TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence`, `internal/senlin/list_request_test.go:TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth`, `internal/senlin/list_request_test.go:TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence`.
- Inherited request-control names cannot become vendor query keys through either public query helpers or custom config injection. SDK-owned authentication/version/transport headers, malformed numeric selections, latest, stale explicit source version headers and foreign arguments fail before list HTTP.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities`, `api/clustering_list_request_test.go:TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts`, `internal/senlin/list_request_test.go:TestListRequestPreflightKeepsKindAndRejectsControlBypasses`.
- Per-call list settings retain the once-resolved canonical cluster and apply only to child list requests; the parent resolution and later scoped Get continue using source settings.
  Tests: `api/clustering_list_request_test.go:TestClusteringListRequestBindingScopeAndOtherOperationsUnchanged`.

Python/Go differences:

- Go confirms and fixes a canonical parent through explicit ID/Name references instead of Python _proxy.py:1078-1081 extracting the supplied cluster ID and interpolating it directly.
- Senlin16.0.0 ClusterPolicyController.index whitelists enabled, policy_name, policy_type and sort. Go rejects initial limit/marker even through Resources and follows only same-origin, same-parent collection links preserving initial filters; advertised deployment markers remain allowed. This intentionally does not simulate unsupported stock-server pagination.
- WithListQuery is a protected, explicit deployment extension and reaches the server; pinned Resource.list discards unknown query keys. Go rejects body/foreign-argument options and accepts owned local-filter and per-call request namespaces through WithListFilter and concrete-query collisions; stock Senlin vendor support is not promised.
- Go requires a cluster_policies array, separate valid canonical raw identities and a matching parent, preserving unknown/null/raw JSON fields. Python Resource.list wraps non-array collections, removes self and overlays URI fields on row dictionaries; Go stores fixed URIClusterID separately and uses PolicyID only as the collection identity callback.
- Published filters use concrete typed inputs and grammar-checked sort, while null/omission and exact numbers remain visible in Body rather than being hidden by Python Resource bool/dict descriptor conversions.
- Go list controls require a non-negative integer MaxItems and optional bool Paginated; max_items/paginated are local option names and cannot be emitted through raw query extensions. Caps count wire rows before local filters, own their state per iteration and do not turn an upper consumption bound into a guaranteed result count. Unlike Python Resource.list max_items, the binding cap sends no limit hint because the audited stock controller whitelists only enabled/policy_name/policy_type/sort.
- At an exact page boundary Python Resource.list may issue a continuation GET before checking max_items at the next raw row; the Go Senlin iterator terminates immediately after its cap and does not validate or request an unused continuation.
- Go separates concrete server query inputs from explicit WithListFilter raw Body inputs rather than automatically partitioning Python keyword arguments. Query-capable fields can also be selected for local filtering; only known pinned fields and owned namespaces are accepted.
- Raw JSON comparison keeps exact decimal numbers and distinguishes bool from number; object filters match recursive subsets of nonempty objects, while arrays remain ordered complete values. Missing scalar fields and explicit null compare as null without descriptor coercion or cached Resource mutation.
- Go exposes WithListHeader and WithListMicroversion separately from query and Body inputs; it preserves SDK ownership of authentication, service-token, authorization and transport/version headers rather than copying an unrestricted Python caller header map.
- A numeric per-call selection is explicit and does not negotiate or mutate the shared client. Explicit empty selection resets to server-default1.0; a conflicting source MoreHeaders version is rejected instead of silently removed. Common Resources and parent resolution/Get/Wait do not accept these typed List options.

Unresolved boundaries:

- Map per-call base_path and the deprecated proxy JMESPath filter forwarded by this proxy; tested max_items/paginated controls limit raw rows locally with no stock-unsupported wire-limit hint.
- Compare automatic Body/query classification, descriptor defaults/coercion, Python scalar equality and Resource.id alternate-policy_id behavior. Explicit raw Body filters are implemented but Go validates required binding identity before any filter and keeps JSON booleans distinct from numbers.
- Compare inherited QueryParameters limit/marker transposition and Resource._get_next_link explicit-limit fallback using Resource.id, including short pages and ordinary-id versus alternate-policy_id precedence. Go intentionally rejects initial limit/marker because the pinned server whitelist lacks them.

### `python:clustering/v1/collect_cluster_attrs`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterattributes.API.InCluster`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterattributes.Scope.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterattributes.Scope.All`

- The fixed escaped JSONPath collection requires selected numeric Senlin1.2+, trims the path and rejects empty/None before parent HTTP, confirms the canonical parent once and rechecks the source/version before each collection page.
  Tests: `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesJSONPathRemainsOneFixedSegment`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesMinimumVersionAndPathPreflight`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesParentCanonicalizationAndGateRecheck`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesExactParentNameUsesCanonicalRawField`.
- Both published HTTP202 and Senlin16.0.0 controller/WSGI HTTP200 decode cluster_attributes synchronously. Exact lowercase id/value fields preserve object/array/scalar, huge numbers and null-versus-omitted values with independent raw Body/Header/status; incidental Location/action does not create a Submission or another request.
  Tests: `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesSynchronousCodesAndRawValues`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesCanonicalRawFieldsOwnIdentityAndValue`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesResponseErrorsPreserveEvidence`.
- The list-only iterator is lazy, stops on break, follows only explicit links retaining its fixed parent and escaped path, and checks version changes before another page. Initial queries and fabricated Get/Delete/status capabilities are rejected, preserving cancellation and whole malformed-response evidence.
  Tests: `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesLazyExplicitPagingAndVersionGuard`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesListOnlyCapabilitiesAndQueries`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesResponseErrorsPreserveEvidence`.

Python/Go differences:

- Pinned _proxy.py:630-648 declares **query but forwards only cluster_id and path to _list. Every supplied query/control is ignored by that proxy. Go correspondingly exposes no List/All query options and rejects initial Resources queries; absent query forwarding is not a missing direct collect proxy contract.
- Go performs a parent confirmation request or exact Name search and fixes its canonical raw ID. Python directly interpolates the supplied parent/path URI values. Go trims and escapes the path as one segment and requires selected numeric1.2+ instead of relying on implicit adapter negotiation and server validation.
- The public API-ref says HTTP202, while the audited Senlin16.0.0 router has no cluster_collect success override and WSGI defaults to HTTP200. Both are accepted as synchronous read results, with no fabricated action state and no live-cloud completion claim.
- Go requires the documented array and canonical valid node id and stores fixed URIClusterID/URIPath separately; raw value bytes preserve missing/null and exact numeric precision. Python Resource.list can wrap singleton collections, removes self, overlays URI fields and constructs mutable descriptor-backed Resource objects.
- Go follows only same-origin, same-collection explicit links, retaining the fixed escaped path and detecting invalid continuation evidence. It does not guess node-ID marker fallback or expose arbitrary Resource routes.

Unresolved boundaries:

- The separate inherited ClusterAttr.list interface remains unmapped: paginated=False, explicit limit/marker and Resource.id fallback, max_items raw-row counting/wire limit, and per-call base_path/microversion/headers. These controls are available on Resource.list itself but are ignored when passed through this declared collect proxy, so this is inherited-resource comparison scope rather than missing collect query forwarding (resource.py:2155-2440; _proxy.py:645-648).
- Compare direct inherited ClusterAttr.list Body-filter inputs node_id, inherited id and attr_value, plus constructor node_id/attr_value aliases, including scalar/recursive-subset filtering and constructor alias-to-wire normalization. The collect proxy never forwards such filters; Go currently requires canonical wire id/value and retains separate fixed URI fields instead of Python row URI overlay (cluster_attr.py:16-31; resource.py:2222-2267,2299-2338,917-951).

### `python:clustering/v1/fetch_cluster_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.FetchMetadata`

- The functional metadata read uses the supported cluster GET200 object route, retaining an independent exact raw metadata map and original Cluster Body/Header/status. Explicit omitted, null and empty-object metadata remain distinct through Present/Null flags, canonical lowercase metadata owns the view despite case-folded extensions, and incidental Location never causes another request.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadViewsPreservePresenceAndRawAuthority`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataReadAliasesAndOwnedDelete`.
- Explicit ID references remain supplied controller routes; Name references resolve a canonical lowercase raw id/name once. Invalid input/source/context fails before HTTP, malformed accepted GET metadata preserves the full original response, and native 403/404 errors retain status/body/header with 404 additionally ErrNotFound.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataNameResolutionUsesCanonicalRawFields`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataPreflightRejectsUnsupportedInputs`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadFailuresPreserveWholeResponse`.

Python/Go differences:

- Pinned _proxy.py:384-399 constructs or reuses a Cluster Resource and MetadataMixin.fetch_metadata uses GET /clusters/{id}/metadata, updating that mutable Resource only if the response includes metadata (common/metadata.py:33-46). Senlin16.0.0 router has no cluster metadata subresource route. Go implements the functional read intent through the supported GET /clusters/{id} and returns a fresh MetadataView instead of attempting the absent route.
- Go uses explicit ID/Name references and context, validates exact canonical Name-list identities, fixes an explicit request identity independently of incidental response IDs, and exposes raw precision/presence/evidence instead of Python descriptor and cached-instance behavior. It does not claim inherited Resource/cache compatibility or automatic action completion.
- This is the documented stateless Go read mapping: supported cluster GET projection, explicit Ref/context and independently owned raw response/presence fields. Mutable input-resource identity/cache and Python runtime deprecation warnings are Go policy differences, not a claim that the absent stock-server metadata route exists.

### `python:clustering/v1/get_cluster_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.GetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.FetchMetadata`

- GetMetadata delegates to FetchMetadata and issues an independent supported cluster GET200 with the same raw precision, omission/null distinction and original response evidence. The alias does not reuse a previous read, follow incidental Location or claim a metadata subresource exists.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadViewsPreservePresenceAndRawAuthority`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadFailuresPreserveWholeResponse`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataReadAliasesAndOwnedDelete`.
- Both read names share source validation and explicit controller identity policy; invalid references, nil/foreign clients, symbolic or malformed selected versions and cancellation remain observable before HTTP.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataPreflightRejectsUnsupportedInputs`.

Python/Go differences:

- Pinned _proxy.py:401-417 emits a deprecation warning and calls fetch_cluster_metadata, whose MetadataMixin GET /clusters/{id}/metadata route is absent from the audited Senlin16.0.0 router. Go keeps a compatibility method alias to its supported cluster GET read and does not emit Python's runtime warning.
- The Go alias returns a fresh typed MetadataView with independent raw values and Cluster evidence rather than updating/reusing the supplied Python Resource instance. Explicit ID/Name and context policies match the Go functional FetchMetadata alternative.
- This is the documented stateless Go read mapping: supported cluster GET projection, explicit Ref/context and independently owned raw response/presence fields. Mutable input-resource identity/cache and Python runtime deprecation warnings are Go policy differences, not a claim that the absent stock-server metadata route exists.

### `python:clustering/v1/set_cluster_metadata`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.SetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithMetadataHeader`

- Set snapshots input JSON and protected header-only options before lookup, observes metadata via GET and shallow-merges top-level keys while preserving unrelated exact raw numbers/null/unknown values. One full-object PATCH202 stays on the supplied ID route or the once-resolved canonical Name route, uses the source's current token/version and returns independent Previous/Requested/ResponseCluster/Operation without claiming requested values were applied.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataShallowMergeSnapshotsAndQueuedResult`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataNameResolutionUsesCanonicalRawFields`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataSharesSourceAndLocksResolvedRoute`.
- Nil or empty updates and semantically equal exact JSON perform GET but no PATCH; object order and decimal spelling do not fabricate a change, while boolean/number and nested missing/null remain distinct. Serialization, unsupported body/query/arguments, owned headers, source changes and canceled contexts fail with their original causes before another request or no-op success.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataNoOpsUseExactJSONEquality`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataPreflightRejectsUnsupportedInputs`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataRevalidatesSourceBeforeEveryRequest`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataNoOpAndSourceRecheck`.
- Malformed GET or accepted PATCH envelopes/metadata, missing/foreign/multiple action Location and cancellation after acceptance retain full original HTTP evidence without a resend or action GET. Wrong status and server failures remain native. The RMW concurrency fixture demonstrates that a concurrent key absent from the observed object is not merged into the replacement and queued acceptance does not imply final state.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadFailuresPreserveWholeResponse`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataAcceptedFailuresNeverResendOrFollowLocation`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataCancellationAfterAcceptanceRetainsEvidence`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadModifyWriteDoesNotClaimConcurrencyOrCompletion`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataAcceptedFailuresKeepOriginalEvidence`.

Python/Go differences:

- Pinned _proxy.py:419-437 forwards metadata kwargs to MetadataMixin.set_metadata, which POSTs /clusters/{id}/metadata and assigns submitted metadata to the mutable Resource cache (common/metadata.py:48-70). The audited Senlin16.0.0 router does not register that route. Go uses supported cluster GET plus PATCH /clusters/{id} with {cluster:{metadata:wholeObject}}, matching the server cluster update action's metadata replacement semantics.
- Go returns an observed/requested/accepted MetadataResult with a mandatory validated action Location and Changed meaning accepted replacement only. It shallow-merges a GET snapshot, normalizes omitted/null metadata only as an empty merge base, and suppresses empty/equal writes instead of making an unsupported synchronous metadata POST or treating cached submitted values as server state.
- Input and response snapshots, exact-decimal comparison with strict JSON boolean/number types, explicit ID/Name resolution, context and header ownership are documented Go policies. No extra numeric version gate, conditional-write guarantee, automatic retry, rollback or completion wait is invented.

Unresolved boundaries:

- The functional supported-route alternative is implemented and tested, but synchronous key-wise metadata transport and mutable Resource/cache return semantics are not reproduced. The Go result intentionally separates observed, requested and accepted states; the audited nonexistent routes are not implementation backlog.
- Server-side atomic key merge or CAS across GET/PATCH and queued action application is not provided or proven. Concurrent updates can be lost; no-op is based on the observed snapshot. This is a documented boundary of the implemented functional layer, not a live-cloud atomicity/completion claim or an authorization to invent metadata routes.

### `python:clustering/v1/delete_cluster_metadata`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.DeleteMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithMetadataHeader`

- Nil keys clear the observed metadata with one whole-object empty replacement; a nonnil empty key slice performs no HTTP and fabricates no observation. Selected duplicate or arbitrary string keys are snapshotted before GET, removed together and sent as one PATCH while preserving exact unrelated values. Missing selected keys and already empty/omitted/null metadata perform GET without PATCH.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataSelectedDeleteIsOneOwnedReplacement`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataNoOpsUseExactJSONEquality`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataReadAliasesAndOwnedDelete`.
- Header-only capability and source/context validation applies even to the zero-HTTP empty-key branch and after custom option side effects. Name resolution and explicit ID routes remain fixed, native missing/server errors are not hidden, and an accepted replacement retains required action Location/evidence without a retry or automatic follow-up.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataZeroHTTPDeleteRevalidatesOptionSideEffects`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataPreflightRejectsUnsupportedInputs`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataRevalidatesSourceBeforeEveryRequest`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataAcceptedFailuresNeverResendOrFollowLocation`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataNoOpAndSourceRecheck`.

Python/Go differences:

- Pinned _proxy.py:439-451 iterates selected keys and invokes MetadataMixin.delete_metadata_item's DELETE /clusters/{id}/metadata/{key}; nil keys invoke PUT /clusters/{id}/metadata with an empty object (common/metadata.py:86-92,137-160). Neither metadata subresource route is registered by the audited Senlin16.0.0 router. Go implements deletion intent using supported cluster GET and a single queued whole-object replacement PATCH.
- Go snapshots a concrete string slice, batches selected-key removal, treats absent selected keys as no change, distinguishes nil clear-all from empty-slice zero HTTP and returns a MetadataResult. Python selected-key requests/cache mutation and None return are not reproduced; no per-key endpoint or fabricated completion is added.
- Previous/Requested raw maps and accepted ResponseCluster/Operation remain separate and independent. Empty-slice results have no observation, while other no-ops preserve the GET evidence. Explicit ID/Name, header ownership and terminal source/context rechecks are documented Go behavior.

Unresolved boundaries:

- The functional supported-route alternative is implemented and tested, but synchronous key-wise metadata transport and mutable Resource/cache return semantics are not reproduced. The Go result intentionally separates observed, requested and accepted states; the audited nonexistent routes are not implementation backlog.
- GET/PATCH and later queued action application have no proven atomic key-deletion/CAS contract. Concurrent metadata added after GET can be removed by the full replacement; the result reports accepted intent rather than final application. No retry/rollback or runtime completion guarantee is claimed.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/)
- [16.0.0 router.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)
- [16.0.0 wsgi.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/common/wsgi.py)
- [Create status fixtures](../api/clustering_cluster_create_status_test.go)
- [16.0.0 clusters.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/clusters.py)
- [16.0.0 conductor/service.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/conductor/service.py)
- [16.0.0 cluster_policies.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/cluster_policies.py)
- [cluster policy bindings README](../clustering/v1/clusterpolicies/README.md)
- [16.0.0 engine/actions/cluster_action.py](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/engine/actions/cluster_action.py)
