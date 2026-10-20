# Router Read Records Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindIgnoreMissing`

- Pinned direct find_router (openstack/network/v2/_proxy.py:5570) accepts a string, ignore_missing=True and query. Safe identities use native GET200 /routers/id with the router envelope and native nested model/timestamp extraction. GET203/204, accepted decode and missing result-ID errors remain terminal without fallback.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsGETModelsAndNativeCodes`.
- Compatible GET400/403/404 may fall back, with explicit 404-only/GET-only policies. Logical absence defaults to nil,nil and strict uses ErrNotFound after successful full native absence; LIST403/404 is never ignored. Native list200/204/300 semantics remain distinct from the member getter.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsFallbackQueryAndMissingPolicies`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsAllPagesAndTerminalObservations`.
- Caller raw query is frozen across both GET and LIST with repetitions, explicit nil/empty name precedence, vendor fields and concurrent option reuse. Only LIST receives an absent-key literal name hint. No project/domain filter is inferred; exact ID OR name matching and cross-project/repeated-ID ambiguity consume all native continuation pages, including later HTTP/decode/cycle errors.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsFallbackQueryAndMissingPolicies`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsQuerySnapshotsAndConcurrentReuse`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsAllPagesAndTerminalObservations`.
- Generated leaf and cached Connection.Network.API collections retain the original live provider, token, headers, HTTP middleware, microversion, caller context and reverse ResourceBase. Unsafe names are literal list-only; GET-only unsafe names, nil/canceled contexts and explicit Details/AllProjects/ExtraSpecs true or false fail before HTTP.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsLiveSourceAndCancellation`, `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsUnsafeNamesAndCapabilityPreflight`.
- Ordinary Resources.List/All preserve explicit raw status/fields/shared/vendor query and native paging while MaxItems applies before local name filtering and Paginated(false) stops continuation without a wire limit. Explicit Ref ID/Name lookup remains separate, with no automatic identity defaults or member lookup added to ordinary lists.
  Tests: `api/find_identity_network_extensions_test.go:TestNativeFindIdentityNetworkExtensionsOrdinaryCollectionsQueriesAndControls`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- Exact pinned Router MRO: revision_number shadows NetworkResource with wire revision; ordered canonical/wire updates, project tenant fallback, null/false/zero, tag [] defaults, unknown extensions and untyped dates. Integer/list/dict/bool casts follow Source, with binary64 truthiness and integer casts.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindLiteralRawQuerySeedsAndHTTPHeader`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadSparseMROAliasesAndDescriptorCasts`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadIntegerUnicodeFloatAndExactLocalEquality`.
- Member raw kwargs both fetch and seed Resource; literal escaped one-segment route, None query omission and accepted parse failure/bodyless seed/default preservation. Canonical underscore Header spelling is exact, wire Header spelling case-insensitive; multi-value HTTP If-Match coalesces comma-space before list cast. Resource/Wire/whole Envelope/Header/status are independent snapshots and forged location projects null.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindLiteralRawQuerySeedsAndHTTPHeader`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindHeaderConstructorSpellingAndNullSeeds`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadMemberAndListStatusReceiptBoundaries`.
- Exact server query aliases and canonical precedence, original repeated fields, tag aliases, local canonical Body filters, unknown and Header-only List kwargs discarded before marshal. Nested dictionary falsey/subset/empty semantics, descriptor errors, exact-decimal local numeric equality and constructor collisions preserve whole page proof.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListExactAliasesRawFieldsTagsAndLocalFilters`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListConstructorDecodeAndLocalFilterFailures`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListUnknownsDiscardedAndNilMapComposition`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadIntegerUnicodeFloatAndExactLocalEquality`.
- GET-first Find permits only clean 400/403/404 fallback, adds automatic name hint only when name absent, distinguishes None presence, searches full pages for exact ID/name, defaults ignore_missing true, and keeps later duplicate/forbidden errors terminal.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindCleanFallbackNamePresenceAndMissing`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindWholeSearchRejectsLateDuplicateOrListError`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListPartialIterationAndAllDropLateError`.
- Lazy complete pager, raw cap before filters/zero-limit cap hint, single page/early break, same-origin/same-routers collection/original filters, raw marker unaffected by consumer changes, short-page fallback, binary64 falsey numeric next, and All discarding partial records on late error.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListRawCapSinglePageAndEarlyStop`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListContinuationScopeAndRawMarkerOwnership`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListFalseyNextAndWholePageProof`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListPartialIterationAndAllDropLateError`.
- Completed 200..399 and every actual 400..599 retain real status/body/Header. Malformed accepted shape/UTF8/descriptor/constructor and local filter failures preserve response proof. Physical Read/Close/custom-cancel/Source faults are terminal, including actual faulty404 no fallback and401 public ErrOriginal/ErrReauth causes, zero willing retry/reauth callbacks and one HTTP/Close.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListConstructorDecodeAndLocalFilterFailures`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadMemberAndListStatusReceiptBoundaries`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadPhysicalFaultsNeverInvokeRetryOrReauth`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadSourceMutationDuringReadRemainsTerminalAfterRestore`.
- Concrete JSON options clone maps/bytes/pointers/callback outputs; nil-map callback composition and unknown-before-marshal behavior, original MarshalerError/cause, selected Source capture before callbacks and unsafe identity/control/nil preflight avoid HTTP. Eight independent APIs concurrently reuse immutable options and isolated record evidence.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadOwnedOptionsMarshalCausesAndPreflight`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListUnknownsDiscardedAndNilMapComposition`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadReusableOptionsConcurrentOwnedViews`.
- Find per-call microversion is member-only; fallback all pages use captured selected default. Standalone List override applies every page, including explicit empty omission without Source mutation. Clean retry and live auth token refresh remain available while protocol/condition/version/body injection and protected Header omission preserve first physical rejection.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindVersionOverrideIsMemberOnly`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadRetryVersionAbsenceAndLiveTokenOwnership`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadExplicitListVersionAppliesToEveryPage`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadRetryCannotOmitOrInjectOwnedProtocolHeaders`.

Python/Go differences:

- Go provides SDK-owned concrete IdentityFindOpts and WithIdentityFind... defaults/snapshots. FindIdentity ignores logical absence by default like pinned Python, while existing explicit Ref Find stays strict. Unsafe names are list-only and every returned/consumed model needs a safe nonblank ID.
- Go raw wire Query is used by both GET and LIST and preserves unknown keys and repetitions; Python list query transposition/local Body filtering is not implicit. Router is_admin_state_up/is_distributed/is_ha and SecurityGroup is_shared require explicit wire admin_state_up/distributed/ha/shared, with tag aliases also explicit.
- Neither binding provides typed Details/AllProjects/ExtraSpecs, hidden lookup, list defaults or automatic microversion changes. Caller project filters are preserved without inference; declared native model/extractor/plural_links policies remain intact.
- The audited Router builder bridge preserves whole raw query in ordinary Resources.List/All as well as FindIdentity, including unknown keys, repetitions and explicit empty values. Its public native typed List API remains intact.
- Go accepts string identity and immutable owned JSON records; mutable Python Resource/Connection/cache/Adapter lifecycle and computed location remain separately tracked. RouterReadRecord aliases the existing RouterActionRecord.
- JSON-domain concrete options and deterministic bulk-map ordering replace arbitrary Python runtime kwargs. Deprecated JMESPath, allow_unknown_params and __conflicting_attrs restoration plus SDK-owned constructor/route/session controls are explicitly bounded preflight errors, not full runtime parity.
- General local numeric equality deliberately uses shared exact-decimal JSON rather than all CPython binary64 rounding/nonfinite/custom-object comparisons. Descriptor integer float cast and bool/numeric-next truthiness use binary64; Unicode integer-string tables use Unicode16 and do not emulate configurable Python string digit caps.
- Completed status200..399 is supported; arbitrary adapters final informational100..199 are outside this Go completion profile. Actual400..599 and all physical/Source/cancel causes retain actual receipts.
- Continuations require same origin, same fixed routers collection and original filters. Dictionary link forms and exact versioned /v2.0/routers alias are explicit Go conveniences; foreign/rewrite/cycle failures preserve actual whole-page proof.
- Safe Header controls are explicit Go policy. Per-call Find version affects only member GET and List version applies all pages; explicit empty version omits Header and selected client remains untouched.
- Neutron27 stock get_router permits project ADMIN_OR_PROJECT_READER; distributed/ha attributes have separate ADMIN branches. Sparse user responses retain Source missing null; this implementation does not elevate access or prove live deployment authorization/extensions/backend support.

### `python:network/v2/routers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatus`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.Network`
- `github.com/JSYoo5B/go-openstacksdk.Connection.NetworkV2`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordListPaginated`

- Pinned direct Proxy.routers delegates query to Router Resource.list. Ordinary Routers.Resources.List/All and API.All provide SDK-owned WithFilter/WithFilters classification and explicit WithBodyFilter options without application builders or callback predicates. Leaf and cached Connection.Network().API/NetworkV2 facades share the native client/provider and preserve the existing returned Router model; no handwritten high-level Router collection or public ResourceAdapter is added.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersEntireQueryAndAliasDescriptor`, `api/router_list_filters_test.go:TestRouterListFiltersRawDescriptorsAndNativeProjection`, `connection_router_filters_test.go:TestConnectionRouterFiltersSharedClientAndRawRevision`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterSemanticFilterEmissionKeepsPrivateCollectionAndPriorEightBindings`.
- All eighteen canonical query attributes and twenty-four accepted canonical/wire spellings are classified, including inherited limit/marker/tags and admin/distributed/ha aliases. Query name/status/id are server-only conditions. Ten non-query Body properties remain local, including deprecated tenant_id and the own revision_number descriptor override. project_id response alias does not create a semantic tenant query or local ProjectID fallback.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersEntireQueryAndAliasDescriptor`, `api/router_list_filters_test.go:TestRouterListFiltersRawDescriptorsAndNativeProjection`, `api/router_list_filters_test.go:TestRouterListFiltersLazyPreflightAndNamespaceCollisions`, `connection_router_filters_test.go:TestConnectionRouterFiltersPreflightAndNativeSurfaces`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterPythonFilterManifestKeepsRevisionOverrideQueryAliasesAndNetworkResourceMRO`.
- Semantic bulk maps give canonical query keys precedence by presence over wire aliases, including nil/false/empty values; individual options targeting the same wire key are last-wins. Query scalar arrays preserve repeated values and exact number spelling while skipping null elements; nil/empty arrays omit URL values but keep key presence. Query objects/nested arrays and nine source control names fail lazily. Unknown semantic names are discarded before value validation.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersEntireQueryAndAliasDescriptor`, `api/router_list_filters_test.go:TestRouterListFiltersLazyPreflightAndNamespaceCollisions`, `api/router_list_filters_test.go:TestRouterListFiltersSnapshotsReplacementAndConcurrentReuse`, `resource/filters_test.go:TestSemanticFiltersValidateOnlyFinalSelectedValues`.
- Semantic query values conflict before HTTP with raw query/page-size values for the same wire key and WithName/WithStatus hints. Semantic and explicit Body predicates for the same canonical field conflict even when values agree. Explicit Body filters accept ten raw keys plus revision_number as an alias for revision; sequential alias options are last-wins, while a bulk map containing both names rejects deterministically. Raw query remains an independent wire namespace.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersLazyPreflightAndNamespaceCollisions`, `api/router_list_filters_test.go:TestRouterListFiltersSnapshotsReplacementAndConcurrentReuse`, `connection_router_filters_test.go:TestConnectionRouterFiltersPreflightAndNativeSurfaces`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`.
- Router.revision_number overrides inherited NetworkResource revision_number and compares original lowercase revision, independently of native Router.RevisionNumber decoded from revision_number. Both raw fields can coexist and disagree. Local tenant_id compares its own original presence/null/value and never substitutes native ProjectID. Raw revision is unknown to semantic WithFilter but is allowed by explicit Body filters or literal WithQuery.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersRawDescriptorsAndNativeProjection`, `api/router_list_filters_test.go:TestRouterListFiltersNativeAndIdentitySurfaceIsolation`, `connection_router_filters_test.go:TestConnectionRouterFiltersSharedClientAndRawRevision`, `connection_router_filters_test.go:TestConnectionRouterFiltersPreflightAndNativeSurfaces`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterPythonFilterManifestKeepsRevisionOverrideQueryAliasesAndNetworkResourceMRO`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterBodyFilterNativeContractRejectsModelBuilderDecoderAndSourceDrift`.
- The complete local lane uses one response-only boolean, two exact-integer responses and seven original-JSON fields. Local defaults are None. Missing/null enable_ndp_proxy remain null; zero/empty JSON values become false and nonzero/nonempty values become true, including the string false. Integer responses accept exact integral numbers or signed decimal integer strings; selected bool/fraction/container values fail with their cause. Caller filters are not coerced, so bool/number/string comparisons remain distinct.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersRawDescriptorsAndNativeProjection`, `api/router_list_filters_test.go:TestRouterListFiltersControlsAndNativeWholePageFailures`, `resource/body_record_boolean_test.go:TestBodyRecordBooleanPresenceOwnershipAndErrorCauses`, `resource/body_record_boolean_test.go:TestBodyRecordBooleanKeepsJSONAndIntegerDescriptorContracts`, `internal/jsonfilter/boolean_test.go:TestBooleanJSONResponseTruthinessAndExactNumericZero`, `resource/body_record_test.go:TestBodyRecordFieldNormalizationPresenceAndCauses`.
- Original Gateway/Route objects retain unknown nested keys, exact large JSON numbers, null members and complete array elements for matching after successful native decoding. Arrays compare order, length and complete elements; standalone object filters recursively match subsets of nonempty objects. Missing/null fields match null while empty arrays and empty actual objects retain their distinct policies. Native returned GatewayInfo/ExternalFixedIP/Route projections, null-to-zero structs, nullable SNAT pointer, availability hint null-to-empty-string and timestamp parsing remain unchanged.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersRawDescriptorsAndNativeProjection`, `api/router_list_filters_test.go:TestRouterListFiltersControlsAndNativeWholePageFailures`, `connection_router_filters_test.go:TestConnectionRouterFiltersSharedClientAndRawRevision`.
- Options snapshot maps/slices/raw JSON and support independent concurrent reuse. Semantic bulk replacement/clear changes only its namespace and may replace an earlier invalid value; explicit Body filters and raw query remain separate. Returned model mutation cannot alter stored options or later native responses. Lazy invalid/unsupported options and canceled contexts emit errors before HTTP instead of silently truncating an iterator.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersSnapshotsReplacementAndConcurrentReuse`, `api/router_list_filters_test.go:TestRouterListFiltersLazyPreflightAndNamespaceCollisions`, `api/router_list_filters_test.go:TestRouterListFiltersControlsAndNativeWholePageFailures`, `connection_router_filters_test.go:TestConnectionRouterFiltersPreflightAndNativeSurfaces`.
- WithMaxItems caps raw rows before local Body/name/status comparisons without refilling matches or inferring a wire limit. First-page mode and consumer break stop before continuation. Native extraction validates the whole current page, including malformed known/nested fields beyond the cap, before raw pairing. Only consumed selected raw fields are normalized; cap/break can avoid an unconsumed raw integer error. Consumed null rows with Body filters reject, query-only null rows retain native zero models, and empty object rows remain valid.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersControlsAndNativeWholePageFailures`, `api/router_list_filters_test.go:TestRouterListFiltersPagingLiveSourceAndNativeStatus`.
- Native RouterPage retains 200/204/300, routers_links with its last rel-next, non-JSON204 empty completion and JSON-empty204 EOF. Top-level next/links.next do not gain new native support. Duplicate matching rows stay duplicated; later HTTP/decode failures, cycles, foreign continuation following and cancellation are terminal, and All returns nil rather than a partial slice on error. Live provider tokens, configured headers/microversion, middleware and service ResourceBase prefixes remain intact across pages.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersPagingLiveSourceAndNativeStatus`, `api/router_list_filters_test.go:TestRouterListFiltersControlsAndNativeWholePageFailures`, `connection_router_filters_test.go:TestConnectionRouterFiltersSharedClientAndRawRevision`.
- Raw WithQuery preserves literal revision/revision_number/tenant_id/status/vendor keys in both list lanes. WithStatus enables the existing case-insensitive native Status predicate whose target is the final last-wins status query, including a later raw option; raw status alone adds no local predicate. Typed API.List, Get, FindIdentity and Router actions keep their independent native contracts and do not acquire semantic list conditions or additional GET/action requests.
  Tests: `api/router_list_filters_test.go:TestRouterListFiltersNativeAndIdentitySurfaceIsolation`, `api/router_list_filters_test.go:TestRouterListFiltersPagingLiveSourceAndNativeStatus`, `connection_router_filters_test.go:TestConnectionRouterFiltersPreflightAndNativeSurfaces`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterSemanticFilterEmissionKeepsPrivateCollectionAndPriorEightBindings`.
- The semantic descriptor requires seven live pinned Python file SHAs and forty-one independent AST anchors proving exact C3 inheritance, effective descriptor defaults/types/aliases, the own revision override, tag/query policies, bool None handling, conversion and direct Proxy controls. Generation requires fresh AST extraction to match the checked-in manifest. Native gates cover twenty-three receiver-qualified reachable declarations, resourcePath and NoZ layout constants, full Router/ListOpts/nested/result/page/builder/decoder shapes and actual compiled type graphs. The extraction chain has no ExtractIntoStructPtr dependency; prior eight semantic bindings/manifests remain unchanged. Hashing source branches does not itself implement inherited cache/session/continuation behavior.
  Tests: `internal/cmd/sdkgen/router_filters_test.go:TestRouterPythonFilterManifestKeepsRevisionOverrideQueryAliasesAndNetworkResourceMRO`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterBodyFilterNativeContractRejectsModelBuilderDecoderAndSourceDrift`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterSemanticFilterEmissionKeepsPrivateCollectionAndPriorEightBindings`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterPythonFilterPinnedSourceRequiresRevisionOverrideAndCompleteClassificationProof`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterRawDependencyLoaderPreservesNativeTimeCodesAndNumbers`, `internal/cmd/sdkgen/router_filters_test.go:TestRouterBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods`.
- Exact pinned Router MRO: revision_number shadows NetworkResource with wire revision; ordered canonical/wire updates, project tenant fallback, null/false/zero, tag [] defaults, unknown extensions and untyped dates. Integer/list/dict/bool casts follow Source, with binary64 truthiness and integer casts.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindLiteralRawQuerySeedsAndHTTPHeader`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadSparseMROAliasesAndDescriptorCasts`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadIntegerUnicodeFloatAndExactLocalEquality`.
- Member raw kwargs both fetch and seed Resource; literal escaped one-segment route, None query omission and accepted parse failure/bodyless seed/default preservation. Canonical underscore Header spelling is exact, wire Header spelling case-insensitive; multi-value HTTP If-Match coalesces comma-space before list cast. Resource/Wire/whole Envelope/Header/status are independent snapshots and forged location projects null.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindLiteralRawQuerySeedsAndHTTPHeader`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindHeaderConstructorSpellingAndNullSeeds`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadMemberAndListStatusReceiptBoundaries`.
- Exact server query aliases and canonical precedence, original repeated fields, tag aliases, local canonical Body filters, unknown and Header-only List kwargs discarded before marshal. Nested dictionary falsey/subset/empty semantics, descriptor errors, exact-decimal local numeric equality and constructor collisions preserve whole page proof.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListExactAliasesRawFieldsTagsAndLocalFilters`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListConstructorDecodeAndLocalFilterFailures`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListUnknownsDiscardedAndNilMapComposition`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadIntegerUnicodeFloatAndExactLocalEquality`.
- GET-first Find permits only clean 400/403/404 fallback, adds automatic name hint only when name absent, distinguishes None presence, searches full pages for exact ID/name, defaults ignore_missing true, and keeps later duplicate/forbidden errors terminal.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindCleanFallbackNamePresenceAndMissing`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindWholeSearchRejectsLateDuplicateOrListError`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListPartialIterationAndAllDropLateError`.
- Lazy complete pager, raw cap before filters/zero-limit cap hint, single page/early break, same-origin/same-routers collection/original filters, raw marker unaffected by consumer changes, short-page fallback, binary64 falsey numeric next, and All discarding partial records on late error.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListRawCapSinglePageAndEarlyStop`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListContinuationScopeAndRawMarkerOwnership`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListFalseyNextAndWholePageProof`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListPartialIterationAndAllDropLateError`.
- Completed 200..399 and every actual 400..599 retain real status/body/Header. Malformed accepted shape/UTF8/descriptor/constructor and local filter failures preserve response proof. Physical Read/Close/custom-cancel/Source faults are terminal, including actual faulty404 no fallback and401 public ErrOriginal/ErrReauth causes, zero willing retry/reauth callbacks and one HTTP/Close.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListConstructorDecodeAndLocalFilterFailures`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadMemberAndListStatusReceiptBoundaries`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadPhysicalFaultsNeverInvokeRetryOrReauth`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadSourceMutationDuringReadRemainsTerminalAfterRestore`.
- Concrete JSON options clone maps/bytes/pointers/callback outputs; nil-map callback composition and unknown-before-marshal behavior, original MarshalerError/cause, selected Source capture before callbacks and unsafe identity/control/nil preflight avoid HTTP. Eight independent APIs concurrently reuse immutable options and isolated record evidence.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadOwnedOptionsMarshalCausesAndPreflight`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadListUnknownsDiscardedAndNilMapComposition`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadReusableOptionsConcurrentOwnedViews`.
- Find per-call microversion is member-only; fallback all pages use captured selected default. Standalone List override applies every page, including explicit empty omission without Source mutation. Clean retry and live auth token refresh remain available while protocol/condition/version/body injection and protected Header omission preserve first physical rejection.
  Tests: `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadFindVersionOverrideIsMemberOnly`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadRetryVersionAbsenceAndLiveTokenOwnership`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadExplicitListVersionAppliesToEveryPage`, `network/v2/extensions/layer3/routers/python_read_contracts_test.go:TestPythonRouterReadRetryCannotOmitOrInjectOwnedProtocolHeaders`.

Python/Go differences:

- Go uses concrete WithFilter/WithFilters and explicit Body options with frozen values rather than Python keyword arguments and mutable Resource objects. Semantic property names and raw response keys remain separate: revision_number reads revision, deprecated tenant_id stays local and project_id stays query-mapped. Explicit Body bulk aliases reject; semantic bulk query canonical keys win by presence, while individual options are last-wins.
- Selected values come from original JSON after native whole-page decoding. Unknown Gateway/Route keys, numeric precision, timestamp spelling and null array members can be matched without changing native model projections. Known native strings/bools/ints/time/nested fields may reject first, including malformed rows beyond a raw cap. Body-filter null rows reject while query-only native zero rows remain accepted.
- Response-only boolean truthiness preserves missing/null and uses exact decimal zero/nonzero; tiny nonzero values remain true where Python float parsing can underflow. Caller conditions retain their types, so Go boolean/number equality differs from Python. Exact-integral response policy rejects fraction/bool/container coercion and accepts signed integer strings; Python can truncate floats, preserve bool-as-int equality or return zero for unsupported numeric strings.
- Python list descriptors wrap non-list values and dict descriptors coerce nonnull nonobjects to empty dictionaries. The Go raw lane does neither; arrays compare complete ordered elements and standalone dictionaries use recursive subset matching only on nonempty objects. Original timestamp spelling is compared even when instants are equivalent.
- Known query arrays become repeated URL values rather than arbitrary Python container formatting. Nil/empty arrays preserve option-key presence while omitting values. Unknown semantic keys are discarded; raw WithQuery remains an explicit wire extension with no automatic alias/body classification.
- Raw local max_items omits Python Resource.list limit hints. Native linked-page codes, full-page decoding, JSON-empty204 EOF and advertised foreign continuation behavior remain unchanged. No per-call base-path/session/header/microversion or deprecated JMESPath field condition is inferred from reserved source controls.
- Go accepts string identity and immutable owned JSON records; mutable Python Resource/Connection/cache/Adapter lifecycle and computed location remain separately tracked. RouterReadRecord aliases the existing RouterActionRecord.
- JSON-domain concrete options and deterministic bulk-map ordering replace arbitrary Python runtime kwargs. Deprecated JMESPath, allow_unknown_params and __conflicting_attrs restoration plus SDK-owned constructor/route/session controls are explicitly bounded preflight errors, not full runtime parity.
- General local numeric equality deliberately uses shared exact-decimal JSON rather than all CPython binary64 rounding/nonfinite/custom-object comparisons. Descriptor integer float cast and bool/numeric-next truthiness use binary64; Unicode integer-string tables use Unicode16 and do not emulate configurable Python string digit caps.
- Completed status200..399 is supported; arbitrary adapters final informational100..199 are outside this Go completion profile. Actual400..599 and all physical/Source/cancel causes retain actual receipts.
- Continuations require same origin, same fixed routers collection and original filters. Dictionary link forms and exact versioned /v2.0/routers alias are explicit Go conveniences; foreign/rewrite/cycle failures preserve actual whole-page proof.
- Safe Header controls are explicit Go policy. Per-call Find version affects only member GET and List version applies all pages; explicit empty version omits Header and selected client remains untouched.
- Neutron27 stock get_router permits project ADMIN_OR_PROJECT_READER; distributed/ha attributes have separate ADMIN branches. Sparse user responses retain Source missing null; this implementation does not elevate access or prove live deployment authorization/extensions/backend support.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--router", default="edge")
parser.add_argument("--mode", choices=("find", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        value = conn.network.find_router(args.router, ignore_missing=False)
        print(json.dumps(value.to_dict(), indent=2))
    else:
        for value in conn.network.routers(name=args.router, max_items=100):
            print(json.dumps(value.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, mode, identity string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "find":
        value, err := network.Routers.FindRecord(
            ctx, identity, routers.WithRecordFindIgnoreMissing(false),
        )
        if err != nil || value == nil { return nil, err }
        return value.Resource, nil
    case "list":
        values, err := network.Routers.AllRecords(
            ctx, routers.WithRecordListQuery("name", identity),
            routers.WithRecordListMaxItems(100),
        )
        if err != nil { return nil, err }
        rows := make([]*resource.RawResource, 0, len(values))
        for _, value := range values { rows = append(rows, value.Resource) }
        return rows, nil
    default:
        return nil, fmt.Errorf("unknown mode %q", mode)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    identity := flag.String("router", "edge", "router name or literal ID")
    mode := flag.String("mode", "list", "find or list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *identity)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## References

- [Install](../../docs/install.md)
- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py#L177)
