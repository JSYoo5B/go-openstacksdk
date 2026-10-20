# Python List Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ListRecordValues`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListJMESPath`

- List sends GET ports with network_id, admin_state_up, repeated security_groups and fixed_ips and binding:host_id, and follows ports_links with the next href.
  Tests: `network/v2/ports/python_parity_core_test.go:TestPythonParityCorePortListQueryAndLinks`.
- Default bodyless /ports GET and Accept header, Source binding/DNS/security/QoS/unknown fields, missing defaults and actual page evidence.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsDefaultRequestAndExtensionFields`.
- Canonical and wire server aliases, repeated Requests values, unknown kwargs discarded before JSON encoding, nested local declared Body filters.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsQueryAliasesAndLocalBodyFilters`.
- Plural/HTTP link continuation, short-page limit/marker fallback, paginated false and raw row cap.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPagingLinksAndLimitMarker`.
- Accepted representations and rejected statuses retain ports operation context; nil option performs no HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsStatusesAndOperationContext`.
- Cloud search uses dict insertion-order pushdown, exact/glob ID-name matching, nested filters and arbitrary full JMESPath; unknown filter fails after real inventory.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSelectionAndPushdown`.
- Explicit version and headers apply to every actual list page.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsHeadersAndMicroversionOnEveryPage`.
- Partial field queries do not seed responses; Source constructor alias order/null/boolean/int/list/dict projection and independent actual Wire/Envelope/Header ownership.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPartialFieldsCoercionAndEvidenceIsolation`.
- Owned runtime controls and malformed UTF-8/JSON fail before HTTP; unknown kwargs discarded without running a JSON Marshaler.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsControlsDiscardAndValidationBeforeEncoding`.
- Option factories/callback maps own input snapshots and nil Headers are initialized for direct custom assignments.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsOptionsOwnMapsAndNilHeaders`.
- Option failure retains simultaneous selected-client mutation or cancellation cause, and no HTTP escapes the boundary.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListAndSearchRecordOptionsGuardCancellationAndSource`.
- Raw cap counts filtered rows, lazy consumer break and single-page stop avoid next page requests.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCapCountsFilteredRowsAndLazyConsumer`.
- CPython float underflow next is falsey; malformed envelope/row/constructor controls and invalid next retain actual accepted page receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsContinuationTruthinessAndTerminalEvidence`.
- 18 accepted/rejected status plus Read/Close/mixed faults close once, invoke no retry/reauth hook, preserve actual receipt and native401 pointer ErrOriginal/ErrReauth fields.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`.
- Clean401 token renewal and clean503 retry retain selected headers/version/bodyless route/query and leave original client configuration intact.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCleanReauthenticationAndRetryKeepPolicy`.
- Eager explicit expression supports scalar/array/object/null values and preserves all original Inventory on expression failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordValuesArbitraryJSONAndInventory`.
- Cloud falsey filters including underflow act as absent; constructor headers snapshot input and unknown dict keys retain actual inventory on local failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsFalseyFiltersHeaderSnapshotAndUnknownKeys`.
- Original generator projections/index/type return null, literals/identity equality return values, typed functions/serialization/syntax fail with zero HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsSourceGeneratorExpressionsDoNoHTTP`.
- Source missing/falsey/nonstring controls retain opaque generator; @/not_null/to_array preserve same lazy generator, HTTP starts only on iteration.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsPreserveReturnedAndNestedGenerators`.
- Source formal header/version/base_path normalization and ordered single-pass conflicting attrs restoration; restored formal duplicates fail before HTTP and restored JMES cannot replace the original expression.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsNormalizeSourceFormalsAndConflictingAttrs`.
- Source unknown non-Body control names and Header kwargs are discarded before JSON marshaling without introducing new query or transport behavior.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsSourceUnknownControlNamesDiscardBeforeMarshaler`.
- Typed/raw/snapshot base_path selects service-relative collection path for every page; unsafe foreign host/query/fragment/traversal fails before HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsBasePathOwnsRouteAndLocksContinuation`.
- Source-profile local filters preserve CPython float rounding, arbitrary integer precision, nested boolean/number equality, underflow and overflow comparison.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsLocalEqualityUsesSourceLoadedFloats`.
- Cloud dict filters preserve list_ports polymorphic Proxy result: null/type errors perform no HTTP, @ retains actual inventory and empty literal containers return no fabricated receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsHonorsProxyJMESBeforeCloudLocalSelection`.
- Raw Source formals and dedicated concrete factories use last option consistently for selected path, headers and microversion.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsFormalFactoriesAndTypedOptionsUseLastValue`.
- Actual Source canonical cloud dict lookup succeeds, wire Body keys fail after real inventory, raw Connection location is snapshotted and applied before selection without modifying Wire.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSourceWireFiltersAndConnectionLocationBeforeSelection`.
- Cloud second-phase SelectLoaded preserves Source numeric matches from first-phase list filtering rather than reversing rounded float/integer or nested underflow comparisons.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSecondFilterKeepsSourceLoadedNumericMatch`.
- Configured CloudLocation is cloned and computed per actual resource before filtering; foreign project clears scope names, keeps original project/zone, and never fabricates Wire location.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsCloudLocationOwnsForeignProjectAndZone`.

Python/Go differences:

- Python transposes is_admin_state_up, is_port_security_enabled, security_group_ids and tenant_id; Go callers use the wire names through ListOpts or WithListQuery.
- Go ListOpts sends tags filters as one comma-separated value, while Python repeats list values.
- Python filters keyword arguments that are Body attributes but not QueryParameters locally after each page; Go sends WithListQuery keys to the server and the caller filters the returned rows.
- The native ports.Port model has no binding:*, dns_*, extra_dhcp_opts, port_security_enabled, qos_*, resource_request, trunk_details or several other Python Body fields, so those values are not returned and cannot be filtered by the caller.
- Python merges the next link's query into its own query parameters; Go requests the next href exactly as Neutron returned it.
- Concrete Go records expose owned declared Resource view and independent actual Wire/Envelope/Header/StatusCode. Full mutable Python Resource/cache/dirty-state/runtime subclass handling remains tracked shared SDK-R1/C1/S1, not counted as generated transport parity.
- ProxyListRecords reproduces Source generator-based JMES zero-HTTP values/null/type errors/opaque identity and nested generators. ListRecordValues is explicitly eager Go expression functionality, not claimed identical to Source generator input.
- ListRecords keeps a resource-only iterator and directs polymorphic truthy expressions to ProxyListRecords; falsey/nonstring Proxy controls preserve lazy records. Standard Go iter.Seq2 iteration semantics remain the owned SDK abstraction.
- List routes remain within captured service endpoint; continuation preserves selected origin/collection/filter and rejects unsafe route cycles. Go dictionary/RFC link compatibility and strict validation are explicit Source differences.
- Both list local filters and cloud dictionary selection opt into CPython JSON-loaded equality. Full JMESPath retains documented exact-decimal/lexical object traversal/strict grammar boundaries, and JSON-map object functions can differ from Python Port exact-typename function checks.

### `python:cloud/_network/NetworkCloudMixin/list_ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListPorts`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListPortValues`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ListSourceValues`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListFilters`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListLocation`

- Default bodyless /ports GET and Accept header, Source binding/DNS/security/QoS/unknown fields, missing defaults and actual page evidence.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsDefaultRequestAndExtensionFields`.
- Canonical and wire server aliases, repeated Requests values, unknown kwargs discarded before JSON encoding, nested local declared Body filters.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsQueryAliasesAndLocalBodyFilters`.
- Plural/HTTP link continuation, short-page limit/marker fallback, paginated false and raw row cap.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPagingLinksAndLimitMarker`.
- Accepted representations and rejected statuses retain ports operation context; nil option performs no HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsStatusesAndOperationContext`.
- Cloud search uses dict insertion-order pushdown, exact/glob ID-name matching, nested filters and arbitrary full JMESPath; unknown filter fails after real inventory.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSelectionAndPushdown`.
- Explicit version and headers apply to every actual list page.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsHeadersAndMicroversionOnEveryPage`.
- Partial field queries do not seed responses; Source constructor alias order/null/boolean/int/list/dict projection and independent actual Wire/Envelope/Header ownership.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPartialFieldsCoercionAndEvidenceIsolation`.
- Owned runtime controls and malformed UTF-8/JSON fail before HTTP; unknown kwargs discarded without running a JSON Marshaler.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsControlsDiscardAndValidationBeforeEncoding`.
- Option factories/callback maps own input snapshots and nil Headers are initialized for direct custom assignments.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsOptionsOwnMapsAndNilHeaders`.
- Option failure retains simultaneous selected-client mutation or cancellation cause, and no HTTP escapes the boundary.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListAndSearchRecordOptionsGuardCancellationAndSource`.
- Raw cap counts filtered rows, lazy consumer break and single-page stop avoid next page requests.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCapCountsFilteredRowsAndLazyConsumer`.
- CPython float underflow next is falsey; malformed envelope/row/constructor controls and invalid next retain actual accepted page receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsContinuationTruthinessAndTerminalEvidence`.
- 18 accepted/rejected status plus Read/Close/mixed faults close once, invoke no retry/reauth hook, preserve actual receipt and native401 pointer ErrOriginal/ErrReauth fields.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`.
- Clean401 token renewal and clean503 retry retain selected headers/version/bodyless route/query and leave original client configuration intact.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCleanReauthenticationAndRetryKeepPolicy`.
- Eager explicit expression supports scalar/array/object/null values and preserves all original Inventory on expression failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordValuesArbitraryJSONAndInventory`.
- Cloud falsey filters including underflow act as absent; constructor headers snapshot input and unknown dict keys retain actual inventory on local failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsFalseyFiltersHeaderSnapshotAndUnknownKeys`.
- Original generator projections/index/type return null, literals/identity equality return values, typed functions/serialization/syntax fail with zero HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsSourceGeneratorExpressionsDoNoHTTP`.
- Source missing/falsey/nonstring controls retain opaque generator; @/not_null/to_array preserve same lazy generator, HTTP starts only on iteration.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsPreserveReturnedAndNestedGenerators`.
- Source formal header/version/base_path normalization and ordered single-pass conflicting attrs restoration; restored formal duplicates fail before HTTP and restored JMES cannot replace the original expression.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsNormalizeSourceFormalsAndConflictingAttrs`.
- Source unknown non-Body control names and Header kwargs are discarded before JSON marshaling without introducing new query or transport behavior.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsSourceUnknownControlNamesDiscardBeforeMarshaler`.
- Typed/raw/snapshot base_path selects service-relative collection path for every page; unsafe foreign host/query/fragment/traversal fails before HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsBasePathOwnsRouteAndLocksContinuation`.
- Source-profile local filters preserve CPython float rounding, arbitrary integer precision, nested boolean/number equality, underflow and overflow comparison.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsLocalEqualityUsesSourceLoadedFloats`.
- Cloud dict filters preserve list_ports polymorphic Proxy result: null/type errors perform no HTTP, @ retains actual inventory and empty literal containers return no fabricated receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsHonorsProxyJMESBeforeCloudLocalSelection`.
- Raw Source formals and dedicated concrete factories use last option consistently for selected path, headers and microversion.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsFormalFactoriesAndTypedOptionsUseLastValue`.
- Actual Source canonical cloud dict lookup succeeds, wire Body keys fail after real inventory, raw Connection location is snapshotted and applied before selection without modifying Wire.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSourceWireFiltersAndConnectionLocationBeforeSelection`.
- Cloud second-phase SelectLoaded preserves Source numeric matches from first-phase list filtering rather than reversing rounded float/integer or nested underflow comparisons.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSecondFilterKeepsSourceLoadedNumericMatch`.
- Configured CloudLocation is cloned and computed per actual resource before filtering; foreign project clears scope names, keeps original project/zone, and never fabricates Wire location.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsCloudLocationOwnsForeignProjectAndZone`.
- Connection uses cached Neutron service; owned per-row CurrentLocation is applied before local selection without changing actual Wire, and callbacks execute once.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsCachedServiceAndAutomaticPerRowLocation`.
- Clean absent catalog returns nonnil empty list before list options; search still evaluates loaded local filters and expressions on [], including scalar/null and invalid expressions, with zero HTTP. Mixed catalog errors and cancellation remain failures.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsCatalogAbsencePrecedesListOptionsButSearchStillEvaluatesEmpty`, `connection_network_record_lists_test.go:TestConnectionPortRecordsMixedCatalogFailureAndCancellationDoNotClaimEmpty`.
- Invalid Connection/context fails before discovery; late-page failure preserves actual response error evidence and prevents a fabricated successful selection.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsPreflightAndLatePageFailureRetainActualEvidence`.
- Source list() on literal arrays, objects, Unicode strings and empty iterables preserves JSON values without fake PortRecord or receipt; JSON object key order is explicitly Go lexical.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListLiteralIterablesDoNotFabricateResources`.
- Null/scalar noniterable and opaque-generator JMES function/syntax failures return before HTTP; result JSON is never fabricated.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListNonIterablesAndGeneratorErrorsDoNoHTTP`.
- Direct returned Source generator is eagerly collected into actual records with unknown/default fields, query/header pushdown, remote-only seed, actual page proof and computed per-record cloud location.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListReturnedGeneratorUsesActualRowsAndNoSeed`.
- Nested generator pointer remains opaque and lazy; serialization fails, explicit later Records iteration performs actual transport.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListNestedGeneratorRemainsOpaqueAndLazy`.
- Missing and CPython falsey JSON filters become empty kwargs; truthy nonobject filters fail before HTTP instead of being treated as cloud search expressions.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListFiltersFalseyAndTruthyNonMappings`.
- Raw filters/dedicated headers/typed callback snapshots own input state and nil callback maps; dedicated headers last-win without empty default overriding Source raw headers.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListHeadersInputAndCallbackSnapshots`.
- A later page physical fault preserves only observed actual Inventory; successful cloud Values/Ports/JSON are absent on failure.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListLateFaultRetainsOnlyActualInventory`.
- Clean absent network service returns nonnil [] before filter expansion or option callbacks; endpoint errors and context remain errors.
  Tests: `connection_port_values_test.go:TestConnectionListPortValuesAbsentServiceSkipsFilterExpansion`.
- Whole Connection shares cached network client and injects current location for actual records, preserves literal values without HTTP/fake records, and retains existing typed ListPorts compatibility.
  Tests: `connection_port_values_test.go:TestConnectionListPortValuesCachedClientLocationAndTypedListCompatibility`.

Python/Go differences:

- Concrete Go records expose owned declared Resource view and independent actual Wire/Envelope/Header/StatusCode. Full mutable Python Resource/cache/dirty-state/runtime subclass handling remains tracked shared SDK-R1/C1/S1, not counted as generated transport parity.
- ProxyListRecords reproduces Source generator-based JMES zero-HTTP values/null/type errors/opaque identity and nested generators. ListRecordValues is explicitly eager Go expression functionality, not claimed identical to Source generator input.
- ListRecords keeps a resource-only iterator and directs polymorphic truthy expressions to ProxyListRecords; falsey/nonstring Proxy controls preserve lazy records. Standard Go iter.Seq2 iteration semantics remain the owned SDK abstraction.
- List routes remain within captured service endpoint; continuation preserves selected origin/collection/filter and rejects unsafe route cycles. Go dictionary/RFC link compatibility and strict validation are explicit Source differences.
- Both list local filters and cloud dictionary selection opt into CPython JSON-loaded equality. Full JMESPath retains documented exact-decimal/lexical object traversal/strict grammar boundaries, and JSON-map object functions can differ from Python Port exact-typename function checks.
- Connection models clean service-catalog absence and explicit endpoint overrides; the broad Python has_service configuration-disabled service policy remains a shared SDK-C1 contract. Typed Connection.ListPorts intentionally retains record-only results; the named list_ports row remains unresolved for polymorphic iterable conversion.
- Concrete JSON/context/captured-client API; arbitrary Python Resource/session/class/callable substitutions are not accepted. Shared cloud configuration disabled-service semantics and mutable Resource R1 contracts remain separate tracked scope.
- Public Gophercloud testhelper/nativefixture reuse and actual response errors preserve physical read/close causes; opaque or literal local JSON never receives invented resource/HTTP evidence.
- Captured route/header/version protection, Go error returns and retained observed inventory differ intentionally from dynamic Python exception-only return conventions.
- Python literal object list() uses insertion order; Go JMES maps use lexical key order. Existing strict JMES syntax/exact numeric evaluator/custom Resource function type differences remain documented.
- Existing typed Connection.ListPorts remains resource-only compatibility API; the complete Source polymorphic list path is explicitly Connection.ListPortValues/API.ListSourceValues.
- Computed current cloud location is injected by Connection and optional for direct leaf; shared Source service configuration disable controls are not inferred from absent endpoint tests.

### `python:cloud/_network/NetworkCloudMixin/search_ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.SearchRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchExpression`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchCloudLocation`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchPorts`

- Default bodyless /ports GET and Accept header, Source binding/DNS/security/QoS/unknown fields, missing defaults and actual page evidence.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsDefaultRequestAndExtensionFields`.
- Canonical and wire server aliases, repeated Requests values, unknown kwargs discarded before JSON encoding, nested local declared Body filters.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsQueryAliasesAndLocalBodyFilters`.
- Plural/HTTP link continuation, short-page limit/marker fallback, paginated false and raw row cap.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPagingLinksAndLimitMarker`.
- Accepted representations and rejected statuses retain ports operation context; nil option performs no HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsStatusesAndOperationContext`.
- Cloud search uses dict insertion-order pushdown, exact/glob ID-name matching, nested filters and arbitrary full JMESPath; unknown filter fails after real inventory.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSelectionAndPushdown`.
- Explicit version and headers apply to every actual list page.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsHeadersAndMicroversionOnEveryPage`.
- Partial field queries do not seed responses; Source constructor alias order/null/boolean/int/list/dict projection and independent actual Wire/Envelope/Header ownership.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPartialFieldsCoercionAndEvidenceIsolation`.
- Owned runtime controls and malformed UTF-8/JSON fail before HTTP; unknown kwargs discarded without running a JSON Marshaler.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsControlsDiscardAndValidationBeforeEncoding`.
- Option factories/callback maps own input snapshots and nil Headers are initialized for direct custom assignments.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsOptionsOwnMapsAndNilHeaders`.
- Option failure retains simultaneous selected-client mutation or cancellation cause, and no HTTP escapes the boundary.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListAndSearchRecordOptionsGuardCancellationAndSource`.
- Raw cap counts filtered rows, lazy consumer break and single-page stop avoid next page requests.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCapCountsFilteredRowsAndLazyConsumer`.
- CPython float underflow next is falsey; malformed envelope/row/constructor controls and invalid next retain actual accepted page receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsContinuationTruthinessAndTerminalEvidence`.
- 18 accepted/rejected status plus Read/Close/mixed faults close once, invoke no retry/reauth hook, preserve actual receipt and native401 pointer ErrOriginal/ErrReauth fields.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`.
- Clean401 token renewal and clean503 retry retain selected headers/version/bodyless route/query and leave original client configuration intact.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCleanReauthenticationAndRetryKeepPolicy`.
- Eager explicit expression supports scalar/array/object/null values and preserves all original Inventory on expression failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordValuesArbitraryJSONAndInventory`.
- Cloud falsey filters including underflow act as absent; constructor headers snapshot input and unknown dict keys retain actual inventory on local failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsFalseyFiltersHeaderSnapshotAndUnknownKeys`.
- Original generator projections/index/type return null, literals/identity equality return values, typed functions/serialization/syntax fail with zero HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsSourceGeneratorExpressionsDoNoHTTP`.
- Source missing/falsey/nonstring controls retain opaque generator; @/not_null/to_array preserve same lazy generator, HTTP starts only on iteration.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsPreserveReturnedAndNestedGenerators`.
- Source formal header/version/base_path normalization and ordered single-pass conflicting attrs restoration; restored formal duplicates fail before HTTP and restored JMES cannot replace the original expression.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsNormalizeSourceFormalsAndConflictingAttrs`.
- Source unknown non-Body control names and Header kwargs are discarded before JSON marshaling without introducing new query or transport behavior.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsSourceUnknownControlNamesDiscardBeforeMarshaler`.
- Typed/raw/snapshot base_path selects service-relative collection path for every page; unsafe foreign host/query/fragment/traversal fails before HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsBasePathOwnsRouteAndLocksContinuation`.
- Source-profile local filters preserve CPython float rounding, arbitrary integer precision, nested boolean/number equality, underflow and overflow comparison.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsLocalEqualityUsesSourceLoadedFloats`.
- Cloud dict filters preserve list_ports polymorphic Proxy result: null/type errors perform no HTTP, @ retains actual inventory and empty literal containers return no fabricated receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsHonorsProxyJMESBeforeCloudLocalSelection`.
- Raw Source formals and dedicated concrete factories use last option consistently for selected path, headers and microversion.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsFormalFactoriesAndTypedOptionsUseLastValue`.
- Actual Source canonical cloud dict lookup succeeds, wire Body keys fail after real inventory, raw Connection location is snapshotted and applied before selection without modifying Wire.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSourceWireFiltersAndConnectionLocationBeforeSelection`.
- Cloud second-phase SelectLoaded preserves Source numeric matches from first-phase list filtering rather than reversing rounded float/integer or nested underflow comparisons.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSecondFilterKeepsSourceLoadedNumericMatch`.
- Configured CloudLocation is cloned and computed per actual resource before filtering; foreign project clears scope names, keeps original project/zone, and never fabricates Wire location.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsCloudLocationOwnsForeignProjectAndZone`.
- Connection uses cached Neutron service; owned per-row CurrentLocation is applied before local selection without changing actual Wire, and callbacks execute once.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsCachedServiceAndAutomaticPerRowLocation`.
- Clean absent catalog returns nonnil empty list before list options; search still evaluates loaded local filters and expressions on [], including scalar/null and invalid expressions, with zero HTTP. Mixed catalog errors and cancellation remain failures.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsCatalogAbsencePrecedesListOptionsButSearchStillEvaluatesEmpty`, `connection_network_record_lists_test.go:TestConnectionPortRecordsMixedCatalogFailureAndCancellationDoNotClaimEmpty`.
- Invalid Connection/context fails before discovery; late-page failure preserves actual response error evidence and prevents a fabricated successful selection.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsPreflightAndLatePageFailureRetainActualEvidence`.

Python/Go differences:

- Concrete Go records expose owned declared Resource view and independent actual Wire/Envelope/Header/StatusCode. Full mutable Python Resource/cache/dirty-state/runtime subclass handling remains tracked shared SDK-R1/C1/S1, not counted as generated transport parity.
- ProxyListRecords reproduces Source generator-based JMES zero-HTTP values/null/type errors/opaque identity and nested generators. ListRecordValues is explicitly eager Go expression functionality, not claimed identical to Source generator input.
- ListRecords keeps a resource-only iterator and directs polymorphic truthy expressions to ProxyListRecords; falsey/nonstring Proxy controls preserve lazy records. Standard Go iter.Seq2 iteration semantics remain the owned SDK abstraction.
- List routes remain within captured service endpoint; continuation preserves selected origin/collection/filter and rejects unsafe route cycles. Go dictionary/RFC link compatibility and strict validation are explicit Source differences.
- Both list local filters and cloud dictionary selection opt into CPython JSON-loaded equality. Full JMESPath retains documented exact-decimal/lexical object traversal/strict grammar boundaries, and JSON-map object functions can differ from Python Port exact-typename function checks.
- Connection models clean service-catalog absence and explicit endpoint overrides; the broad Python has_service configuration-disabled service policy remains a shared SDK-C1 contract. Typed Connection.ListPorts intentionally retains record-only results; the named list_ports row remains unresolved for polymorphic iterable conversion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--network", default="")
    parser.add_argument("--host", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {}
    if args.network:
        query["network_id"] = args.network
    if args.host:
        query["binding_host_id"] = args.host
    for port in conn.network.ports(**query):
        print(json.dumps(port.to_dict(), ensure_ascii=False))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    networkID := flag.String("network", "", "server network_id filter")
    host := flag.String("host", "", "local binding_host_id filter")
    mode := flag.String("mode", "list", "list, proxy, eager or search")
    expression := flag.String("expression", "[*].id", "JMESPath expression")
    name := flag.String("name", "", "search name, ID or glob")
    filters := flag.String("filter", "", "search raw JSON filters")
    flag.Parse()
    if err := run(*cloud, *networkID, *host, *mode, *expression, *name, *filters); err != nil { log.Fatal(err) }
}

func run(cloud, networkID, host, mode, expression, name, filters string) error {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    api := network.Ports
    var options []ports.RecordListOption
    if networkID != "" { options = append(options, ports.WithRecordListQuery("network_id", networkID)) }
    if host != "" { options = append(options, ports.WithRecordListQuery("binding_host_id", host)) }
    switch mode {
    case "list":
        for row, err := range api.ListRecords(ctx, options...) {
            if err != nil { return err }
            if err := printPort(row); err != nil { return err }
        }
    case "proxy":
        options = append(options, ports.WithRecordListQuery("jmespath_filters", expression))
        result, err := api.ProxyListRecords(ctx, options...)
        if err != nil { return err }
        if generator, ok := result.Value.(*ports.RecordListGenerator); ok {
            for row, err := range generator.Records {
                if err != nil { return err }
                if err := printPort(row); err != nil { return err }
            }
        } else if result.JSON != nil {
            fmt.Println(string(result.JSON))
        } else {
            fmt.Printf("opaque result: %T\n", result.Value)
        }
    case "eager":
        result, err := api.ListRecordValues(ctx, expression, options...)
        if err != nil { return err }
        fmt.Println(string(result.Value))
    case "search":
        var search []ports.RecordSearchOption
        if filters != "" { search = append(search, ports.WithRecordSearchFilters(json.RawMessage(filters))) }
        result, err := api.SearchRecords(ctx, name, search...)
        if err != nil { return err }
        fmt.Println(string(result.Value))
    default:
        return fmt.Errorf("unknown mode %q", mode)
    }
    return nil
}

func printPort(row *ports.PortRecord) error {
    data, err := json.MarshalIndent(row.Resource.Body, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py)
- [Source `Proxy._list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Port](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py#L347)
