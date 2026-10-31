# Clusterpolicies

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/wait_for_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.Scope.WaitForStatus`

- Required nonempty status uses unlimited SDK time, ERROR-only failures and2-second intervals by default, with case-insensitive target precedence and caller last-wins failure/attribute/timeout policy. Four status models and five explicit string-attribute models are verified; default unsupported attributes fail before HTTP.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`.
- Typed polling uses the selected Senlin source, exact reverse-proxy path and current provider token for each GET; explicit IDs remain fixed even when response IDs differ. Name lookup is exact, happens once and preserves missing/duplicate/invalid-ID/later-page failures before detail polling. Neither waiter submits DELETE or follows action/channel/Location URLs.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`, `connectiontest/connection_senlin_wait_test.go:TestConnectionSenlinWaitersShareLiveTokenVersionAndFixedRoutes`.
- Default and overridden deadlines are observed on actual outgoing HTTP contexts, without sleeping for120seconds. Parent cancellation/deadlines are respected; callbacks report zero for nonterminal Senlin models, stop at completion/failure/HTTP errors, and callback cancellation prevents another GET. Existing Resources.Wait retains its5-minute common policy.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`.
- ClusterPolicy polling fixes one canonical parent and direct policy identity, uses PolicyID rather than binding ID for Name lookup, rejects default status before HTTP, and supports GET-only deletion polling without detaching a policy. Exact lower-case raw identities/names own routes and each request rechecks the service source.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesPreflightAndScopedWaits`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactBodyIdentityOwnsRoutes`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactNameIgnoresCaseFoldedExtensions`, `connectiontest/connection_senlin_scopes_test.go:TestConnectionSenlinParentScopesShareLiveAuthVersionAndFixedRoutes`.

Python/Go differences:

- Go uses explicit Ref and new typed results rather than cached mutable Python Resource input. IDs begin with a fresh GET; Names begin with the server list lookup result. WaitForDelete returns error instead of retaining the original Resource pointer.
- Go requires a nonempty string target, positive duration intervals and exported typed string attributes. Python nullable target/attribute and interval=None/zero normalization are distinct; unknown raw Body attributes or nested attribute paths are not inferred.
- Go passes deadlines into HTTP requests and gives cancellation priority instead of checking timeout only between fetches. Published status/envelope rules preserve exact Body/Header/StatusCode and reject null/malformed success instead of permissive falsy completion.
- BuildInfo has no new singleton polling route: pinned proxy get uses requires_id=False, while inherited wait fetch uses its normal requires_id=True and a normal ID-less BuildInfo cannot be fetched that way. Services remain list-only rather than inventing GET/delete/status capabilities.
- Python returns the passed Resource immediately, with no request, when its cached attribute already equals the target (wait_for_status), and otherwise fetches with skip_cache=True; Go always begins with a fresh request.
- Python wait_for_delete returns the original Resource when a fetch is missing or its status equals deleted; Go WaitForDelete returns only an error.
- Python accepts any Resource subclass and an arbitrary attribute name; Go provides waiters for the typed Senlin models and exported string attributes only, so arbitrary or custom Resource inputs have no equivalent.
- Python callback receives the progress attribute or 0; Senlin models have no progress field and Go reports zero.

### `python:clustering/v1/wait_for_delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.Scope.WaitForDelete`

- Deletion defaults to120-second SDK timeout. Four declared status models complete on deleted or404, while five statusless models complete on404 and ignore incidental raw status. No delete capability is fabricated and403/409/native/transport/malformed200 remain errors.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`.
- Typed polling uses the selected Senlin source, exact reverse-proxy path and current provider token for each GET; explicit IDs remain fixed even when response IDs differ. Name lookup is exact, happens once and preserves missing/duplicate/invalid-ID/later-page failures before detail polling. Neither waiter submits DELETE or follows action/channel/Location URLs.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`, `connectiontest/connection_senlin_wait_test.go:TestConnectionSenlinWaitersShareLiveTokenVersionAndFixedRoutes`.
- Default and overridden deadlines are observed on actual outgoing HTTP contexts, without sleeping for120seconds. Parent cancellation/deadlines are respected; callbacks report zero for nonterminal Senlin models, stop at completion/failure/HTTP errors, and callback cancellation prevents another GET. Existing Resources.Wait retains its5-minute common policy.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`.
- ClusterPolicy polling fixes one canonical parent and direct policy identity, uses PolicyID rather than binding ID for Name lookup, rejects default status before HTTP, and supports GET-only deletion polling without detaching a policy. Exact lower-case raw identities/names own routes and each request rechecks the service source.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesPreflightAndScopedWaits`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactBodyIdentityOwnsRoutes`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactNameIgnoresCaseFoldedExtensions`, `connectiontest/connection_senlin_scopes_test.go:TestConnectionSenlinParentScopesShareLiveAuthVersionAndFixedRoutes`.

Python/Go differences:

- Go uses explicit Ref and new typed results rather than cached mutable Python Resource input. IDs begin with a fresh GET; Names begin with the server list lookup result. WaitForDelete returns error instead of retaining the original Resource pointer.
- Go requires a nonempty string target, positive duration intervals and exported typed string attributes. Python nullable target/attribute and interval=None/zero normalization are distinct; unknown raw Body attributes or nested attribute paths are not inferred.
- Go passes deadlines into HTTP requests and gives cancellation priority instead of checking timeout only between fetches. Published status/envelope rules preserve exact Body/Header/StatusCode and reject null/malformed success instead of permissive falsy completion.
- BuildInfo has no new singleton polling route: pinned proxy get uses requires_id=False, while inherited wait fetch uses its normal requires_id=True and a normal ID-less BuildInfo cannot be fetched that way. Services remain list-only rather than inventing GET/delete/status capabilities.
- Python returns the passed Resource immediately, with no request, when its cached attribute already equals the target (wait_for_status), and otherwise fetches with skip_cache=True; Go always begins with a fresh request.
- Python wait_for_delete returns the original Resource when a fetch is missing or its status equals deleted; Go WaitForDelete returns only an error.
- Python accepts any Resource subclass and an arbitrary attribute name; Go provides waiters for the typed Senlin models and exported string attributes only, so arbitrary or custom Resource inputs have no equivalent.
- Python callback receives the progress attribute or 0; Senlin models have no progress field and Go reports zero.

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

Status: `go_mapping`.

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
- Python Proxy._list also accepts a per-call base_path and the deprecated jmespath_filters (it warns RemovedInSDK60Warning) among its keyword arguments; the Go routes are SDK-owned and neither option is exposed.
- Python Resource.list validates keywords with allow_unknown_params=True: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. Go sends explicit extension queries to the server through WithListQuery, applies Body filters only through the explicit WithListFilter with source-validated names, and rejects unknown filter names, reserved names and names owned by typed options before HTTP.
- Python rows are rebuilt with Resource.existing, which applies descriptor coercion and defaults; Go keeps typed snapshots with the raw JSON and filters compare exact raw JSON values without coercion.
- Python cluster_policies passes cluster_id=Resource._get_id(cluster) without resolution (the URL accepts what the server accepts) and declares sort, policy_name, policy_type and is_enabled (sent as enabled); with an explicit limit its fallback marker is Resource.id, the alternate policy_id. Go resolves the cluster once, requires the canonical binding identity before any filter and rejects an initial limit or marker because the audited server whitelist lacks them.
- Go keeps JSON booleans distinct from numbers in raw Body filters and compares exact decimals; Python compares attribute scalars by equality after descriptor coercion.
- The explicit-limit marker fallback, MaxItems limit hints and short-page continuation are verified with HTTP fixtures only; no live Senlin server default or maximum limit was observed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
binding = conn.clustering.get_cluster_policy("POLICY_NAME_OR_ID", "CLUSTER_NAME_OR_ID")
for binding in conn.clustering.cluster_policies("CLUSTER_NAME_OR_ID", is_enabled=False):
    print(binding.id, binding.policy_id)
```

```go
api := clusterpolicies.New(client)
scope, err := api.InCluster(ctx, resource.ID("CLUSTER_NAME_OR_ID"))
if err != nil { return err }
binding, err := scope.Get(ctx, "POLICY_NAME_OR_ID")
if err != nil { return err }
fmt.Println(binding.ID, binding.PolicyID, binding.ClusterID, binding.URIClusterID)

for binding, err := range scope.List(ctx, clusterpolicies.WithListEnabled(false)) {
    if err != nil { return err }
    fmt.Println(binding.PolicyName, binding.IsEnabled)
}
```

```go
bindings, err := scope.All(ctx,
    clusterpolicies.WithListEnabled(false),
    clusterpolicies.WithListPolicyName("POLICY_NAME"),
    clusterpolicies.WithListPolicyType("senlin.policy.scaling-1.0"))
if err != nil { return err }
fmt.Println(len(bindings))
```

```go
err = scope.WaitForDelete(ctx, resource.ID("POLICY_NAME_OR_ID"),
    resource.WithTimeout(30*time.Second),
    resource.WithPollInterval(time.Second))
if err != nil { return err }
```

```go
listingAPI := clusterpolicies.New(client)
listingScope, err := listingAPI.InCluster(ctx, resource.ID("CLUSTER_ID"))
if err != nil { return err }
values, err := listingScope.All(ctx, clusterpolicies.WithListMaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingScope.List(ctx, clusterpolicies.WithListPaginated(false)) {
    if err != nil { return err }
    fmt.Println(value.PolicyID)
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func FilterClusterPolicies(ctx context.Context, conn *sdk.Connection, cluster resource.Ref) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	scope, err := service.ClusterPolicies.InCluster(ctx, cluster)
	if err != nil {
		return err
	}
	for binding, err := range scope.List(ctx,
		clusterpolicies.WithListFilter("is_enabled", false),
		clusterpolicies.WithListFilter("data", map[string]any{"team": "infra"}),
		clusterpolicies.WithListMaxItems(50), clusterpolicies.WithListPaginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(binding.ID, binding.PolicyID, binding.URIClusterID)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#cluster-policies)
- [16.0.0 cluster policy controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/cluster_policies.py)
- [16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
