# Python Cloud Parity

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_identity/IdentityCloudMixin/create_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Create`

- create_service(name, type=..., description=...) is POST /v3/services with name, type, enabled true and description, and a missing type is sent as JSON null through CreateOpts.Extra like Python's explicit type=None.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityServiceHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python accepts type or service_type and sends type null when both are missing; Go needs CreateOpts.Extra{"type": nil} to send null because the typed Type field is always encoded.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/update_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Update`

- update_service finds the service by ID then name (GET /v3/services/<x>, then GET /v3/services?name=<x>) and sends PATCH /v3/services/<id> with only the requested fields.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityServiceHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.find does the ID GET and list fallback in one call and also falls back after GET 400 or 403; Go chains Find(resource.ID(x), resource.WithIgnoreMissing()) and Find(resource.Name(x), resource.WithIgnoreMissing()), which falls through only after a 404.
- The Python list match compares the ID or the name and raises DuplicateResource; the Go name lookup compares only the name and returns resource.ErrAmbiguous.
- Python raises SDKException when the service is missing; Go returns a nil row from the chain and the caller must not call Update.
- Python maps service_type to type; Go callers set UpdateOpts.Type.
- Python commits only attributes that differ from the found Resource and skips the PATCH when nothing changed; Go Update sends every field the caller sets, so an unchanged value is still sent.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/list_services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.List`

- list_services() is one GET /v3/services without query and decodes every row.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityServiceHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/search_services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewServiceReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.ServiceReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.ServiceReadCloud.SearchServices`

- search_services lists with the same GET /v3/services as list_services before filtering locally.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityServiceHelpers`.
- Exact cloud search services has no query kwargs: queryless list materializes every page before identifier exact/glob and ordered local filtering. Duplicate IDs remain and selected ServiceRecord/header/view is a detached clone of physical Inventory.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudSearchQuerylessAndLocalGlobSubset`.
- Source falsey filters preserve declared rows/null name Python None comparison; truthy string JMES uses materialized rows and returns arbitrary JSON independently of typed Services/actual Inventory. Source loaded numeric nested mapping equality and ordered short circuit are reused.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudSearchStringExpressionsUseMaterializedRows`, `identity/service_record_cloud_test.go:TestPythonServiceCloudFalseyFiltersPreserveDeclaredRows`, `identity/service_record_cloud_test.go:TestPythonServiceCloudOrderedMismatchHidesUnreachedBadShape`.
- Reached missing/nested-shape/invalid expression errors follow all pages, keep actual Inventory and never fabricate ResponseError. Late HTTP failure occurs before local filtering and retains native status/header/body/method/URL plus prior row proof.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudLocalErrorsFollowEveryPageAndKeepInventory`, `identity/service_record_cloud_test.go:TestPythonServiceCloudLateHTTPFailurePrecedesInvalidLocalFilter`.
- Cloud immutable filters/headers/location preflight captures original source before options; current location is computed before filtering, ignores undeclared remote ownership, and owner drift/caller custom cancellation/nil options prevent later callbacks and HTTP.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudLocationBeforeFilterAndOwnedArguments`, `identity/service_record_cloud_test.go:TestPythonServiceCloudOptionsGuardCallbacksAndSelectedSourceSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Concrete library-owned immutable JSON descriptors/options/results and captured client/context replace mutable Python Resource/session/subclass objects. Native typed Service/List/All/Find/CRUD contracts are preserved; the SDK user implements no builders or decoder interfaces.
- Direct factory SourceNoConnection has default location:null; explicit immutable CloudLocation/current scope is a Go convenience, and root Connection bridges supply selected identity client plus current location. Whole SDK-R1/C1/S1 mutable Resource/cache/Connection/session/discovery lifecycle remains tracked separately; these three mappings do not claim full SDK supported parity.
- Go preserves actual view/wire/page/header/status evidence and error causes instead of mutable Python object identity or process-global warnings. Protected fixed route/collection/filter/header policy, strict UTF-8/nonnegative pagination/cycle detection and HTTP Link interoperability intentionally bound broader Source adapters and Requests uri/url-link behavior.
- Go JSON marshalling/map-backed option snapshots use deterministic lexical object key order where Python dict has insertion order; ordered WithQuery/raw dictionary filter members retain caller order. Pure JMES JSON/number/syntax policies are concrete Go mappings rather than every Python runtime object/function type behavior.
- Source warnings/deprecations are not process-global output. SearchServices returns an explicit ServiceSearchResult with polymorphic JSON and physical Inventory instead of a nominal Python list that may actually be scalar/object/string/null. No expression result receives invented ServiceRecord/HTTP metadata.

### `python:cloud/_identity/IdentityCloudMixin/get_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewServiceReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.ServiceReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.ServiceReadCloud.GetService`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordResolveOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordResolveHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordResolveMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordResolveLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.ResolveRecord`

- get_service(name_or_id) with the default filters=None sends GET /v3/services/<x>, falls back to GET /v3/services?name=<x> after 404 and returns nil when nothing matches.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityServiceHelpers`.
- Omitted or literal JSON null filters use the Source closed-signature finder: member GET100..399 success, only clean400/403/404 fallback to name hint and exact ID-or-name, missing nil and duplicate ambiguity. Fallback consumes after first match but stops at second; no raw arbitrary fetch kwargs are exposed by this named Source call.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordClosedFinderMemberAndCleanStatusFallback`, `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordMissingDuplicatesAndGetOnlyVersion`.
- Get default fallback preserves exact name rather than glob, accepted original member/list record proof and missing JSON null. {}false/0/[]/empty-string are non-None filters and use queryless search; filtered selection honors glob/dictionaries and never deduplicates repeated IDs.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudGetNullFilterMemberVersusAllOtherFalseySearchBranches`, `identity/service_record_cloud_test.go:TestPythonServiceCloudGetDefaultFallbackIsLiteralNameNotGlob`, `identity/service_record_cloud_test.go:TestPythonServiceCloudGetDeprecatedFiltersDuplicatesSearchGlobAndMissing`.
- Deprecated expression output follows actual Python truth/len/integer-index control: falsey returns None; >1 raises ambiguity; one-element array or one-codepoint string selects JSON; truthy scalar len and one-member object integer-key indexing raise local errors with actual Inventory only.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudGetDeprecatedExpressionTruthLengthAndIntegerIndex`.
- Explicit GET-only MV override does not change fallback LIST selected version. Closed resolve option/header/location snapshot, nil/source/cancellation guard, physical403 close failure without fallback or retries preserve native evidence and caller causes.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordMissingDuplicatesAndGetOnlyVersion`, `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordPhysicalForbiddenDoesNotFallThrough`, `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordOptionsSnapshotCancellationAndExplicitLocation`.
- Get options execute once using captured selected source before callbacks. Explicit immutable current location remains independent of raw project/location; deprecated complete-list lateHTTP beats local malformed filters and unreachable invalid filter on empty selection stays unvisited.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudGetOptionCallbackOnceCapturedSourceAndOwnedLocation`, `identity/service_record_cloud_test.go:TestPythonServiceCloudGetLateHTTPBeforeDeprecatedLocalErrorsAndEmptySelection`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.find does the ID GET and list fallback in one call and also falls back after GET 400 or 403; Go chains Find(resource.ID(x), resource.WithIgnoreMissing()) and Find(resource.Name(x), resource.WithIgnoreMissing()), which falls through only after a 404.
- The Python list match compares the ID or the name and raises DuplicateResource; the Go name lookup compares only the name and returns resource.ErrAmbiguous.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Concrete library-owned immutable JSON descriptors/options/results and captured client/context replace mutable Python Resource/session/subclass objects. Native typed Service/List/All/Find/CRUD contracts are preserved; the SDK user implements no builders or decoder interfaces.
- Direct factory SourceNoConnection has default location:null; explicit immutable CloudLocation/current scope is a Go convenience, and root Connection bridges supply selected identity client plus current location. Whole SDK-R1/C1/S1 mutable Resource/cache/Connection/session/discovery lifecycle remains tracked separately; these three mappings do not claim full SDK supported parity.
- Go preserves actual view/wire/page/header/status evidence and error causes instead of mutable Python object identity or process-global warnings. Protected fixed route/collection/filter/header policy, strict UTF-8/nonnegative pagination/cycle detection and HTTP Link interoperability intentionally bound broader Source adapters and Requests uri/url-link behavior.
- Go JSON marshalling/map-backed option snapshots use deterministic lexical object key order where Python dict has insertion order; ordered WithQuery/raw dictionary filter members retain caller order. Pure JMES JSON/number/syntax policies are concrete Go mappings rather than every Python runtime object/function type behavior.
- Go ServiceGetResult.Service exists only for actual ordinary selected resources; expression output is JSON Value plus actual Inventory. Missing success is JSON null; failed result Value is absent. DeprecatedFilters records Source filters-is-not-None branch instead of emitting Python process-global warnings.
- RecordResolveOpts headers/MV/location are explicit Go conveniences, not Source get_service/find_service kwargs. The original named closed-signature Cloud method cannot reach raw Resource.fetch**query branches; the generic Source Find/Resource gaps remain separate inventory operations.
- Default closed finder exposes selected record/error proof but not every discarded fallback row in Inventory; deprecated search branch retains actual prior Inventory. Pure Go scalar/index/type errors are wrapped concrete invalid-option/ambiguity errors rather than Python TypeError/KeyError/SDKException classes.

### `python:cloud/_identity/IdentityCloudMixin/delete_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Remove`

- delete_service finds the service, skips the DELETE when nothing matches, and Remove ignores a DELETE 404 like ignore_missing=True while returning a 409.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityServiceHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.find does the ID GET and list fallback in one call and also falls back after GET 400 or 403; Go chains Find(resource.ID(x), resource.WithIgnoreMissing()) and Find(resource.Name(x), resource.WithIgnoreMissing()), which falls through only after a 404.
- The Python list match compares the ID or the name and raises DuplicateResource; the Go name lookup compares only the name and returns resource.ErrAmbiguous.
- Python returns True or False and logs and swallows SDKException from the delete; Go returns nil when the lookup finds nothing (nil row) or the DELETE succeeds or is a 404, and returns other errors such as 409.

### `python:cloud/_identity/IdentityCloudMixin/create_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithCreateField`

- create_endpoint resolves the service by ID or name, then POSTs one /v3/endpoints body per URL with url, interface, service_id, enabled true and region_id (JSON null when region is not given) sent through WithCreateField("region_id", ...).
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityEndpointHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.find does the ID GET and list fallback in one call and also falls back after GET 400 or 403; Go chains Find(resource.ID(x), resource.WithIgnoreMissing()) and Find(resource.Name(x), resource.WithIgnoreMissing()), which falls through only after a 404.
- The Python list match compares the ID or the name and raises DuplicateResource; the Go name lookup compares only the name and returns resource.ErrAmbiguous.
- Python expands public_url, internal_url and admin_url into one create per interface in that order and rejects url or interface combined with them before HTTP; Go callers issue one Create per interface.
- Python raises SDKException when the service is missing; the Go chain returns a nil service.
- Python sends interface null when url is given without interface; Go CreateOpts requires Availability, URL and ServiceID and rejects the request before HTTP.
- CreateOpts.Region encodes the legacy region key, so the Python region_id key needs WithCreateField("region_id", ...).
- Python returns a list of Endpoint Resources; Go returns one endpoint pointer per Create call.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/update_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.WithUpdateField`

- update_endpoint(endpoint_id, ...) is PATCH /v3/endpoints/<id> with service_name_or_id sent as service_id without lookup and region sent as region_id through WithUpdateField.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityEndpointHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python sends only dirty attributes and skips the PATCH when nothing was given; Go sends an empty endpoint envelope for empty options.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/list_endpoints`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.List`

- list_endpoints() is one GET /v3/endpoints without query.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityEndpointHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/search_endpoints`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewEndpointReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.EndpointReadCloud.SearchEndpoints`

- search_endpoints lists with the same GET /v3/endpoints as list_endpoints before filtering locally.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityEndpointHelpers`.
- search_endpoints materializes unfiltered selected v3 Endpoint Source records across all pages before id/name exact-or-glob and ordered local dict/loaded float/bool equality or arbitrary materialized-list JMESPath; falsey filters skip mapping and invalid local filters run after pages. Actual rows preserve Wire/page/header/status, independent selected copies/Inventory; no server filter hint. Library-owned immutable options/header/client/provider/route/version/location source capture and chained REST operation guard preserve native bodyless GET and every physical error/cause without fallback/retry/auth hiding.
  Tests: `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudSearchGlobLocalFiltersAndActualRecords`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudSearchFalseyFiltersAndPythonIdentifiers`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudLocalFilterErrorsFollowEveryPageAndKeepInventory`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudNestedEmptyFilterUsesActualTruthiness`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudJMESUsesMaterializedListAndKeepsNoInventedReceipt`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetNoneBypassesDeprecatedFalseySearch`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetCleanFallbackExactIDNameDuplicatesAndMissing`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetSeedAliasesWireAndExplicitLocation`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetFilteredArbitraryTruthLenAndIndex`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudOwnedHeadersOptionsAndConcurrentReuse`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudPreflightSourceCancelAndOuterGuard`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudLateNativeErrorWinsOverInvalidLocalFilter`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudPhysicalFaultsNeverFallbackRetryOrInventRows`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Concrete Go typed v3 factory/JSON operands and safe owned route/context/header guards constrain arbitrary Python proxy/session/object behavior. Runtime mutable proxy API-version metadata is shared SDK-S1; Resource/Connection mutable lifecycle remains SDK-R1 and optional HTTP cache SDK-C1.
- Source None vs present falsey filters branch is explicit in EndpointReadOpts and DeprecatedFilters; Go exposes branch metadata rather than executing Python warning categories. Source get_endpoint does not use _get_entity and no Resource/dict identity passthrough is invented.
- Separate RawMessage Value, real EndpointRecord and Inventory preserve arbitrary JMESPath output without fabricating HTTP association. Search/deprecated get retains physically read rows on late/local failure; default private find exposes returned record receipt, not an invented complete inventory of its rejected/member/fallback observations.
- SDK-owned explicit CloudLocation uses captured/current scope; direct ServiceClient default is null. Stock Keystone27.0.0 get/list policy is admin_required or role:reader and system_scope:all, with system/project scope declaration; project reader alone and actual deployed cloud acceptance are not claimed.

### `python:cloud/_identity/IdentityCloudMixin/get_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewEndpointReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.EndpointReadCloud.GetEndpoint`

- get_endpoint(id) with the default filters=None sends GET /v3/endpoints/<id> and, after 404, the unfiltered GET /v3/endpoints because Endpoint declares no name query.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityEndpointHelpers`.
- get_endpoint omitted/literal-null filters executes private Source GET-first Resource.find, only clean400403404 fallback to complete exact id-or-name list with default missing nil and duplicate ErrAmbiguous. Every other filter value including falsey enters deprecated materialized search, then loaded truthiness/len/integer index0; arbitrary values and actual records have independent result carriers and no invented receipt. Library-owned immutable options/header/client/provider/route/version/location source capture and chained REST operation guard preserve native bodyless GET and every physical error/cause without fallback/retry/auth hiding.
  Tests: `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudSearchGlobLocalFiltersAndActualRecords`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudSearchFalseyFiltersAndPythonIdentifiers`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudLocalFilterErrorsFollowEveryPageAndKeepInventory`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudNestedEmptyFilterUsesActualTruthiness`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudJMESUsesMaterializedListAndKeepsNoInventedReceipt`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetNoneBypassesDeprecatedFalseySearch`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetCleanFallbackExactIDNameDuplicatesAndMissing`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetSeedAliasesWireAndExplicitLocation`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudGetFilteredArbitraryTruthLenAndIndex`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudOwnedHeadersOptionsAndConcurrentReuse`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudPreflightSourceCancelAndOuterGuard`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudLateNativeErrorWinsOverInvalidLocalFilter`, `identity/endpoint_record_cloud_test.go:TestPythonEndpointCloudPhysicalFaultsNeverFallbackRetryOrInventRows`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.find does the ID GET and list fallback in one call and also falls back after GET 400 or 403; Go chains Find(resource.ID(x), resource.WithIgnoreMissing()) and Find(resource.Name(x), resource.WithIgnoreMissing()), which falls through only after a 404.
- The Python list match compares the ID or the name and raises DuplicateResource; the Go name lookup compares only the name and returns resource.ErrAmbiguous.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Concrete Go typed v3 factory/JSON operands and safe owned route/context/header guards constrain arbitrary Python proxy/session/object behavior. Runtime mutable proxy API-version metadata is shared SDK-S1; Resource/Connection mutable lifecycle remains SDK-R1 and optional HTTP cache SDK-C1.
- Source None vs present falsey filters branch is explicit in EndpointReadOpts and DeprecatedFilters; Go exposes branch metadata rather than executing Python warning categories. Source get_endpoint does not use _get_entity and no Resource/dict identity passthrough is invented.
- Separate RawMessage Value, real EndpointRecord and Inventory preserve arbitrary JMESPath output without fabricating HTTP association. Search/deprecated get retains physically read rows on late/local failure; default private find exposes returned record receipt, not an invented complete inventory of its rejected/member/fallback observations.
- SDK-owned explicit CloudLocation uses captured/current scope; direct ServiceClient default is null. Stock Keystone27.0.0 get/list policy is admin_required or role:reader and system_scope:all, with system/project scope declaration; project reader alone and actual deployed cloud acceptance are not claimed.

### `python:cloud/_identity/IdentityCloudMixin/delete_endpoint`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints.API.Remove`

- delete_endpoint(id) looks the endpoint up first, skips the DELETE when nothing matches, and otherwise sends DELETE /v3/endpoints/<id> through Remove.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityEndpointHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.find does the ID GET and list fallback in one call and also falls back after GET 400 or 403; Go chains Find(resource.ID(x), resource.WithIgnoreMissing()) and Find(resource.Name(x), resource.WithIgnoreMissing()), which falls through only after a 404.
- Python returns True or False and logs and swallows SDKException from the delete; Go returns nil when the lookup finds nothing (nil row) or the DELETE succeeds or is a 404, and returns other errors such as 409.

### `python:cloud/_identity/IdentityCloudMixin/create_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Create`

- create_domain is POST /v3/domains with name, enabled true and description only when it is given.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityDomainGroupRoleHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/update_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.UpdateDomain`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.FindIdentity`

- With a domain ID no lookup is made and one PATCH carries only supplied attributes; Enabled=false and an empty description are sent.
  Tests: `connectiontest/connection_py_cloud_compute_identity_updates_test.go:TestPyCloudComputeUpdateDomainIDSkipsLookupAndDropsEmptyName`.
- With a name the domain is found by GET then name list, a missing domain returns ErrNotFound with no PATCH, and a call with nothing to change sends no PATCH.
  Tests: `connectiontest/connection_py_cloud_compute_identity_updates_test.go:TestPyCloudComputeUpdateDomainByNameLooksUpAndNoopSendsNothing`.
- A canceled context sends nothing.
  Tests: `connectiontest/connection_py_cloud_compute_identity_updates_test.go:TestPyCloudComputeUpdateGroupMissingAndCanceled`.

Python/Go differences:

- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding.
- Python drops falsy name and description (an empty description cannot be sent); Go omits an empty Name but preserves a non-nil empty Description because the update distinguishes them.
- Python takes domain_id or name_or_id as two strings and raises SDKException when neither is given; Go takes one resource.Ref (ID = domain_id, Name = name_or_id lookup).
- Python's name path uses the cloud get_domain (find_domain ignore_missing=True) and raises SDKException when missing; Go returns resource.ErrNotFound.
- When no attribute is supplied Python builds a Domain from the ID alone and sends nothing; Go returns a Domain with only the ID.
- Python returns a Resource; Go returns the Gophercloud domain model.

### `python:cloud/_identity/IdentityCloudMixin/delete_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.FindIdentity`

- With domain_id, delete_domain sends PATCH /v3/domains/<id> with enabled false and then DELETE /v3/domains/<id> without ignoring 404.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- delete_domain(name_or_id=...) maps to Domains.FindIdentity (GET /domains/<x>, then GET /domains?name=<x>) followed by PATCH /domains/<id> {"domain":{"enabled":false}} and DELETE /domains/<id>; a missing domain sends only the two lookups and a DELETE 404 is returned as an error like ignore_missing=False.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudDeleteDomainByName`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns False when name_or_id finds nothing and re-raises SDKException from the disable or delete; Go returns errors from Update and Delete.
- Python raises SDKException before HTTP when neither domain_id nor name_or_id is given; Go has no combined helper and the caller passes the ID to Update/Delete or the name to FindIdentity.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/list_domains`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewDomainReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.ListDomains`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.SearchDomains`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadLocation`

- Cloud list_domains expands arbitrary keyword mapping into Source query/local filters; enabled aliases and nested options survive local processing and actual yielded inventory preserves response fields.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudListKeywordQueryAliasAndLocalFilters`.
- Cloud eagerly consumes Source Proxy result by Python iterable kind. Literal list/string/object results and nested generators stay HTTP0; opaque generators are not forcibly encoded, JSON exists only for complete JSON data.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudListProxyPolymorphicResultIsUnread`.
- Concrete already-v3 API factory captures Source/header state before callbacks, checks sticky selected API/source/custom cancel before and after options and prevents restore callbacks or fabricated preflight proof.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudOptionsStickySourceCancelAndHeaderSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- SDK-owned concrete record/options API preserves actual Wire/Envelope/metadata without builder interfaces; original native API is unchanged.
- Already selected Keystone v3 API required; no implicit arbitrary-v2 ensure_service_version conversion or automatic Connection current-location bridge. Explicit CloudLocation is cloned and current project retained because Domain declares no project/zone.
- Fixed origin/collection/query/headers/version bounds reject foreign route/traversal/cycles and malformed markers with actual receipt; amounts are nonnegative Go integers.
- Go iter.Seq2 can be reused and repeats captured requests rather than sharing Python generator single-use state. Proxy expression has SDK Source/option preflight before pure result.
- HTTP Link is deliberate Go interoperability for Source next[uri] versus ordinary Requests next[url] shape. JSON rootlink precedence/dict-ignore/null-error follows Source.
- Object-valued Cloud Proxy literal iterates sorted Go keys rather than Python dictionary insertion order. Opaque/nested generator remains Value/ValueObjects and is not forcibly encoded.
- Generic Connection/session GET response caching and arbitrary synchronized Resource lifecycle remain shared SDK-R1/C1 obligations; this finite Go mapping does not claim supported/full SDK parity. Connection.ListDomainRecords/SearchDomainRecords now supply cached service and explicit-or-default location.

### `python:cloud/_identity/IdentityCloudMixin/search_domains`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewDomainReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.ListDomains`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.SearchDomains`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainReadLocation`

- name_or_id presence is preserved with a pointer: nil forwards mapping filters to list_domains; explicit empty/nonempty first materializes unfiltered domains and applies local exact/glob identifier plus mapping/JMESPath. Name/filters stay absent from server in that branch.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudSearchIdentifierPresenceSelectsSourceBranch`.
- Absent identifier kwargs expansion rejects string/false/zero/empty-string/array before HTTP; exactly None filters becomes {}. Identifier-present expression filters run on actual materialized rows and yield arbitrary scalar/null JSON with no fabricated records.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudSearchAbsentKeywordMappingVersusPresentJMESPath`.
- Absent-identifier Proxy nested generator remains an iterable ValueObjects result without invented JSON; public records remain separate from opaque/literal data.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudListProxyPolymorphicResultIsUnread`.
- Local loaded nested subset/glob selection keeps duplicates, CPython rounded float/bool/integer equality and selected record clones independent of physical inventory.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudLocalLoadedSubsetGlobAndIndependentRecord`.
- Complete HTTP materialization precedes invalid local expression, so later503 wins and physical receipt/partial Inventory survives.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudLateHTTPPrecedesLocalFilterError`.
- Explicit cloned location is applied before local selection and trusts configured current project rather than unknown wire extension.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudExplicitLocationOwnedBeforeLocalFiltering`.
- Sticky Source/context options and source header snapshot stop restore callbacks and HTTP on invalid preflight.
  Tests: `identity/domain_record_cloud_test.go:TestPythonDomainCloudOptionsStickySourceCancelAndHeaderSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Without name_or_id Python passes filters to list_domains server-side and does not filter locally.
- SDK-owned concrete record/options API preserves actual Wire/Envelope/metadata without builder interfaces; original native API is unchanged.
- Already selected Keystone v3 API required; no implicit arbitrary-v2 ensure_service_version conversion or automatic Connection current-location bridge. Explicit CloudLocation is cloned and current project retained because Domain declares no project/zone.
- Fixed origin/collection/query/headers/version bounds reject foreign route/traversal/cycles and malformed markers with actual receipt; amounts are nonnegative Go integers.
- Go iter.Seq2 can be reused and repeats captured requests rather than sharing Python generator single-use state. Proxy expression has SDK Source/option preflight before pure result.
- HTTP Link is deliberate Go interoperability for Source next[uri] versus ordinary Requests next[url] shape. JSON rootlink precedence/dict-ignore/null-error follows Source.
- Object-valued Cloud Proxy literal iterates sorted Go keys rather than Python dictionary insertion order. Opaque/nested generator remains Value/ValueObjects and is not forcibly encoded.
- Generic Connection/session GET response caching and arbitrary synchronized Resource lifecycle remain shared SDK-R1/C1 obligations; this finite Go mapping does not claim supported/full SDK parity. Connection.ListDomainRecords/SearchDomainRecords now supply cached service and explicit-or-default location.

### `python:cloud/_identity/IdentityCloudMixin/get_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewDomainReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.DomainReadCloud.GetDomain`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithDomainGetLocation`

- get_domain(domain_id=...) is a direct GET /v3/domains/<id>, and get_domain(name_or_id=...) maps to Domains FindIdentity which sends the GET then the name list.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityDomainGroupRoleHelpers`, `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- DomainID presence wins NameOrID; nonnil direct GET ignores filters completely without marshaler/parse/IO, and records only deprecated warning presence flag. Direct missing404 is an error; nil DomainID selects closed ignore-missing finder and logical absence is nil record/no error.
  Tests: `identity/domain_record_get_test.go:TestPythonCloudDomainGetIDPresenceWinsAndFiltersAreNeverConsumed`, `identity/domain_record_get_test.go:TestPythonCloudDomainGetDirectMissingVersusClosedFinderMissing`.
- Actual Source Resource.find(None)/fetch/prepare_request raises InvalidRequest before HTTP despite cloud doc comment; both identifiers omitted maps to typed preflight error/no invented proof. Empty/numeric Source route edge is deliberately narrowed to safe nonblank Go strings.
  Tests: `identity/domain_record_get_test.go:TestPythonCloudDomainGetBothOmittedMatchesActualNoIDPreflight`.
- Closed cloud-only Domain resolver uses clean400/403/404 GET-first name hint then all-page ID OR name; repeated ID is ambiguous, list199 is accepted, late503 overrides prior match, GET-only version differs from list session default, and existing list profile remains unchanged.
  Tests: `identity/v3/domains/python_record_get_test.go:TestPythonDomainResolveRecordClosedCleanFallbackAbsentDuplicatesAndLateProof`, `identity/v3/domains/python_record_get_test.go:TestPythonDomainResolveRecordGetOnlyVersionAndProfileIsolation`.
- Cloud closed fallback preserves duplicate/late native proof and ignored invalid filters, with no search/expression branch or full Proxy.find_domain mapping growth.
  Tests: `identity/domain_record_get_test.go:TestPythonCloudDomainGetClosedFallbackDuplicatesAndLateHTTPProof`.
- Captured source/header state precedes per-callback cloned options, explicit location is projected with owned configured scope and unknown response location remains Wire; sticky source/custom cancel/nil option creates no HTTP and no fabricated receipt.
  Tests: `identity/domain_record_get_test.go:TestPythonCloudDomainGetOwnOptionLocationHeadersAndSourceSnapshot`, `identity/domain_record_get_test.go:TestPythonCloudDomainGetOptionsStickySourceCancelAndNil`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- The deprecated filters argument is a no-op in Python.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- SDK-owned typed Go transport/location conveniences and raw nullable Domain record preserve actual Wire/Envelope/header/status; existing native Domain Get/Create/Update and full find_domain judgments stay unchanged.
- Go uses required literal nonblank UTF-8 string identity, immutable escaped path and selected native-v3 client. Source empty/numeric/false IDs may request domains/ collection; mutable Resource/dict/object inputs and unsafe raw URL composition are narrowed to documented Go/SDK-C1/S1 boundaries.
- Direct Source actual both-None finder raises no-ID InvalidRequest despite misleading cloud comment; Go ErrInvalidOption preflight preserves HTTP0 instead of pretending None success.
- Ignored filters are not evaluated. Source warnings.warn is represented by DeprecatedFilters presence flag; global Python warning-as-error configuration is not cloned.
- Explicit immutable location is a Go convenience and default standalone view is null; automatic Connection bridge/current scope inference is absent.
- Generic cache/synchronized arbitrary Resource lifecycle/session/Connection remain SDK-R1/C1/S1 differences; finite named get implementation is not full SDK supported. No arbitrary fetch controls or full Proxy.find_domain audit/row promotion.

### `python:cloud/_identity/IdentityCloudMixin/list_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.All`

- list_groups() is GET /v3/groups and list_groups(domain_id=...) adds only the domain_id query.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityDomainGroupRoleHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/search_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewGroupReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.GroupReadCloud.SearchGroups`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithGroupSearchLocation`

- search_groups lists with the same GET /v3/groups (with domain_id when given) as list_groups before filtering locally.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityDomainGroupRoleHelpers`.
- Cloud materializes every Group page with only server domain_id, then exact/glob ID/name and ordered nested loaded local filtering; duplicate rows remain and selected records clone independently of physical inventory.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudSearchDomainServerAndLocalGlobSubset`.
- Truthy string JMESPath sees materialized declared rows after identifier selection and yields arbitrary scalar/null/list JSON with no fabricated record association; falsey filters preserve rows and Python str(None) identifier behavior.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudSearchStringExpressionsUseMaterializedRows`, `identity/group_search_records_test.go:TestPythonGroupCloudFalseyFiltersPreserveDeclaredRows`.
- Every page precedes local-filter evaluation, so invalid expression/unknown key/reached bad shape preserves Inventory after HTTP; ordered mismatch skips bad later shape and late HTTP503 wins over invalid expression.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudLocalErrorsFollowEveryPageAndKeepInventory`, `identity/group_search_records_test.go:TestPythonGroupCloudOrderedMismatchHidesUnreachedBadShape`, `identity/group_search_records_test.go:TestPythonGroupCloudLateHTTPFailurePrecedesInvalidLocalFilter`.
- Explicit location is projected before local selection, unknown response project_id/location cannot replace current scope, and caller filters/domain/headers/location are captured without aliases.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudLocationBeforeFilterAndOwnedArguments`.
- Selected Source/header snapshot precedes options; before/after each callback sticky Source/custom cancellation guards prevent restore callback/HTTP and invalid preflight has no invented receipt.
  Tests: `identity/group_search_records_test.go:TestPythonGroupCloudOptionsGuardCallbacksAndSelectedSourceSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Concrete Go options own request/defaults, nullable raw Resource and independent actual response channels. Existing native List/Get/FindIdentity semantics are unchanged.
- Already selected Keystone v3 API required; no automatic v2 ensure_service_version conversion or Connection current location inference. Explicit immutable CloudLocation retains current project because Group declares no project_id/zone.
- Fixed origin/collection/query/header/version bounds reject foreign URL/traversal/cycle and malformed identities/markers using actual receipt. Paging amounts are nonnegative Go integers.
- iter.Seq2 can be iterated again and repeats captured requests rather than Python generator single-consumption state. Proxy pure expressions still perform SDK option/source preflight before returning.
- Connection cache and generic synchronized Resource object lifecycle remain separately tracked as SDK-R1/C1; finite Group read mapping is not full SDK parity.

### `python:cloud/_identity/IdentityCloudMixin/get_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewGroupReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.GroupReadCloud.GetGroup`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.ResolveRecord`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Default/None uses a closed domain_id-only Source finder: literal GET, only clean400403404 fallback with name/domain hints, exact ID/name, default nil missing, duplicate error, GET-only per-call MV and actual descriptor/Wire/page/header/status evidence. Caller options/domain/header/location bytes snapshot before sticky API/client/provider/route/version/context/private outer guard; physical faults retain native401 ErrOriginal value and all causes with no fallback/auth/retry/backoff hiding. Group search forwards raw domain including None/false/zero and collects every Source row/page before ordered dict/loaded equality or materialized-list JMES; loaded truthiness/len/integer index0 retains false/0/empty selected values and reports duplicate/type errors without invented record receipt.
  Tests: `identity/group_record_get_test.go:TestCloudGetGroupNoneVersusPresentFalseySearch`, `identity/group_record_get_test.go:TestCloudGetGroupLateNativeFailureBeforeLocalFilter`, `identity/group_record_get_test.go:TestCloudGetGroupStickyOptionsCancelAndOuterGuard`, `identity/group_record_get_test.go:TestCloudGetGroupPhysicalCloseSourceCancelKeepActualNativeEvidence`, `identity/group_record_get_test.go:TestCloudGetGroupJMESFirstTruthLenAndIndexKeepsActualInventory`, `identity/group_record_get_test.go:TestCloudGetGroupOwnedRawDomainLocationAndLoadedNestedFilter`, `identity/v3/groups/python_closed_lookup_test.go:TestClosedGroupDomainSeedAndActualSourceEvidence`, `identity/v3/groups/python_closed_lookup_test.go:TestClosedGroupFallbackVersionsExactIDNameAndDefaultMissing`, `identity/v3/groups/python_closed_lookup_test.go:TestClosedGroupOwnedCallbacksRawDomainAndPreflight`, `identity/v3/groups/python_closed_lookup_test.go:TestClosedGroupPhysicalFaultKeepsNative401ValueAndAllCauses`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Generic raw Resource.fetch/header/session kwargs and mutable Resource/Connection/cache/version metadata remain SDK-R1/C1/S1. Full named Proxy find_project/find_group stays unresolved; closed ResolveRecord exposes only identity/domain_id and owned Go Header/MV/location conveniences.
- Go result separates actual Inventory/record from local arbitrary JSON, with nil Value/record for missing and DeprecatedFilters marker instead of Python warning categories. Literal typed identity, safe route/cursor/blank guards and JSON-bounded operands are intentional Go bounds; no object passthrough or fabricated HTTP metadata.
- Source Project truthy string JMES fails at dict(**filters) despite documentation; Group materialized JMES supports arbitrary JSON and Source First shape/errors. Private scalar wrapper is a local common-predicate adaptation and never an HTTP resource.
- Verified Keystone28.0.0 Project member GET allows admin/system/domain reader and own-project user; fallback/list needs admin/system/domain reader. Group member/list allows admin/system reader or target Group domain reader. Deployment overrides and live acceptance are not established by fixtures.

### `python:cloud/_identity/IdentityCloudMixin/create_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Create`

- create_group with a domain sends GET /v3/domains/<domain> (a 404 stops before the POST) and then POST /v3/groups with name, description when truthy and domain_id.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityDomainGroupRoleHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python passes domain to get_domain as domain_id, so only a domain ID resolves and a missing domain raises NotFoundException instead of the documented SDKException; Go Domains.Get returns the 404 error.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/update_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.UpdateGroup`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- The group is found strictly by GET then name list (extra queries such as domain_id pass to both), and only attributes that differ from the found group are sent; an unchanged call sends no PATCH, matching Resource dirty tracking.
  Tests: `connectiontest/connection_py_cloud_compute_identity_updates_test.go:TestPyCloudComputeUpdateGroupSendsOnlyChangedAttributes`.
- A missing group returns ErrNotFound with no PATCH and cancellation sends nothing.
  Tests: `connectiontest/connection_py_cloud_compute_identity_updates_test.go:TestPyCloudComputeUpdateGroupMissingAndCanceled`.

Python/Go differences:

- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding.
- Python forwards update_group **kwargs to the find query; Go passes find queries as resource.IdentityFindOption values.
- Python drops falsy name and description; Go omits an empty Name but compares and sends a non-nil empty Description when it differs.
- Python returns the found Resource unchanged when nothing differs; Go returns a copy of the found model.
- Python Proxy.request can serve the lookup GET from a response cache; Go has no client-side GET cache.

### `python:cloud/_identity/IdentityCloudMixin/delete_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Remove`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Groups, Roles, Projects and Users FindIdentity followed by Remove(resource.ID(id)) send GET /<kind>/<name> and GET /<kind>?name=<name> (with domain_id on both when WithIdentityFindQuery adds it) and then DELETE /<kind>/<id>; 204 and 404 return nil like the Proxy delete default ignore_missing=True, 409 is returned, and a missing row sends no DELETE.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudFindThenRemove`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns True or False and logs and swallows SDKException from the delete; Go returns nil when the lookup finds nothing (nil row) or the DELETE succeeds or is a 404, and returns other errors such as 409.
- delete_group calls find_group without domain_id, so the Go call uses FindIdentity without WithIdentityFindQuery.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/list_roles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewRoleReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.RoleReadCloud.ListRoles`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListQuery`

- Source cloud list_roles expands ordinary Proxy kwargs into the owned list classifier, preserves server name/domain and local options filter, and exposes independent JSON Value/Roles/actual Inventory with203 header/status.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudListForwardsOwnedProxyQueryAndLocalBodyFilters`.
- list() conversion preserves JSON literal lists/string iteration/object key lists and opaque nested generator with no invented HTTP/Role association. Noniterable null/number/bool and opaque array functions fail before HTTP.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudListLiteralOpaqueAndNonIterableBranchesNeverFabricateHTTP`.
- Captured source headers precede list option callbacks. Actual late503 body/header/method/URL preserves prior Inventory while callback runs once; nil/canceled preflight makes zero requests.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudListOptionCallbacksSnapshotSourceAndLateNativeProof`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Existing native Role/List/All/Get/FindIdentity remain unchanged. Concrete functional options and SDK-owned profile descriptors/defaults/decoders avoid caller builders; declared Resource JSON and separate actual Wire/Envelope/header/status are defensively owned.
- Selected Keystone v3 API is required; automatic v2 ensure_service_version conversion is not added. Standalone native client computed location is null; explicit immutable location supplies cloud/region/current project without inference. Role has no project_id/zone descriptors, unknown response facts do not replace that scope and the typed convenience normalizes zone to null.
- Source root link precedence and ignored Keystone dictionary-next are opt-in on this new profile. Fixed origin/path/query/header/version, stronger cycles, representable safe markers and nonnegative Go caps are deliberate Go policies; RFC HTTP Link corrects the pinned Requests next uri/url mismatch.
- Go iterator reuse repeats the captured guarded operation instead of Python generator single-use state. Pure Proxy expressions still run SDK option/source preflight; JSON serialization is separate from opaque/nested generator results.
- SDK-R1/C1/S1 remain separately tracked Connection-wide synchronized Resource/runtime, optional response cache and session/version negotiation. This finite named mapping is not supported full SDK parity or authenticated live-cloud execution. No new find_role mapping or closure of its raw Resource.fetch/control/header kwargs is claimed.
- RoleListResult.Value is []any and JSON is populated only if every value is JSON-representable. Object-valued Proxy expressions convert to deterministically sorted Go keys instead of Python insertion order; literal/opaque values do not acquire synthesized transport evidence.

### `python:cloud/_identity/IdentityCloudMixin/search_roles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity.RoleReadCloud.SearchRoles`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchMicroversion`

- Search lists all pages with only domain query then exact/glob ID/name and ordered recursive loaded-JSON filters. Duplicate records stay in selection, selected records clone away from actual Inventory and203 header/status.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudSearchDomainServerAndLocalGlobSubset`.
- Truthful materialized string JMESPath yields arbitrary JSON and nil Roles with Expression=true. Name is selected first, empty selection is still evaluated, falsey filters retain declared rows and late HTTP fails before invalid local expression.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudSearchStringExpressionsUseMaterializedRows`, `identity/role_record_cloud_test.go:TestPythonRoleCloudFalseyFiltersPreserveDeclaredRows`, `identity/role_record_cloud_test.go:TestPythonRoleCloudLateHTTPFailurePrecedesInvalidLocalFilter`.
- Local missing keys/shape errors occur after actual list and retain Inventory; insertion-order mismatch avoids evaluating later invalid shape. No fake HTTP error is added for purely local selection failures.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudLocalErrorsFollowEveryPageAndKeepInventory`, `identity/role_record_cloud_test.go:TestPythonRoleCloudOrderedMismatchHidesUnreachedBadShape`.
- Explicit typed owned location is applied before local filters and Wire remains original; source callback snapshots/mutations/nil/cancel are guarded throughout the facade and list dependency.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudLocationBeforeFilterAndOwnedArguments`, `identity/role_record_cloud_test.go:TestPythonRoleCloudOptionsGuardCallbacksAndSelectedSourceSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Existing native Role/List/All/Get/FindIdentity remain unchanged. Concrete functional options and SDK-owned profile descriptors/defaults/decoders avoid caller builders; declared Resource JSON and separate actual Wire/Envelope/header/status are defensively owned.
- Selected Keystone v3 API is required; automatic v2 ensure_service_version conversion is not added. Standalone native client computed location is null; explicit immutable location supplies cloud/region/current project without inference. Role has no project_id/zone descriptors, unknown response facts do not replace that scope and the typed convenience normalizes zone to null.
- Source root link precedence and ignored Keystone dictionary-next are opt-in on this new profile. Fixed origin/path/query/header/version, stronger cycles, representable safe markers and nonnegative Go caps are deliberate Go policies; RFC HTTP Link corrects the pinned Requests next uri/url mismatch.
- Go iterator reuse repeats the captured guarded operation instead of Python generator single-use state. Pure Proxy expressions still run SDK option/source preflight; JSON serialization is separate from opaque/nested generator results.
- SDK-R1/C1/S1 remain separately tracked Connection-wide synchronized Resource/runtime, optional response cache and session/version negotiation. This finite named mapping is not supported full SDK parity or authenticated live-cloud execution. No new find_role mapping or closure of its raw Resource.fetch/control/header kwargs is claimed.

### `python:cloud/_identity/IdentityCloudMixin/get_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetRoleRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewRoleReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.RoleReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetDomainIDJSON`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetExpression`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.RoleReadCloud.GetRole`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ResolveRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupDomainIDJSON`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Fixed resolver sends raw nullable/false/zero/empty/repeated/object-key domain query and constructor seed, preserves actual response overwrite/descriptor defaults and independently copied view/wire/page/header/status evidence.
  Tests: `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupDomainSeedRawQueryAndDescriptorEvidence`.
- Source100..399 completion and only clean400/403/404 GET fallback retain name/domain hint, exact ID-or-name selection, missing nil, late page failure and immediate second-match ambiguity. Per-call MV applies GET only; fallback keeps selected source version.
  Tests: `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupCleanStatusFallbackAndLiteralNameHint`, `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupMissingDuplicateLatePageAndGetOnlyMicroversion`.
- Closed rawdomain/header/location opts are immutable; nil/context/custom cancellation/source mutation/private guard block work without fake receipt. Malformed accepted representation retains actual response. Existing profiles/old native behavior are untouched.
  Tests: `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupImmutableOptsSourceCancellationAndExplicitLocation`, `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupPrivateGuardAndMalformedAcceptedReceipt`.
- Physical accepted/rejected/native401 read/close/mixed matrix preserves actual body/header/status and all causes, suppresses every Retry/Reauth/Backoff callback and prevents403/404 fallback; clean renewal/retry keeps frozen domain/header/MV with live token.
  Tests: `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupPhysicalFaultStopsFallbackAndNativeCallbacks`, `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupCleanAuthRetryKeepsFrozenRequest`.
- None/literalnull uses fixed domain-aware finder; non-None falsey filters use search and return its first entity without a final find. Role search always forwards raw nullable/false/zero/empty/repeated domain and does not put nameOrID in server name query.
  Tests: `identity/role_record_get_test.go:TestPythonCloudGetRoleFiltersNoneVersusFalseySearchAndNoFinalFind`, `identity/role_record_get_test.go:TestPythonCloudGetRoleRawFalseyDomainAlwaysForwarded`.
- Role full search and loaded nested/exact/glob predicates keep duplicate IDs, missing and ordered mismatch short circuit, detached actual chosen record; late HTTP precedes local failure and unvisited invalid filters on empty selection remain unvisited.
  Tests: `identity/role_record_get_test.go:TestPythonCloudGetRoleFilteredGlobLoadedMappingDuplicatesAndMissing`, `identity/role_record_get_test.go:TestPythonCloudGetRoleLateHTTPPrecedesLocalErrorAndKeepsActualInventory`.
- Role follows truthiness/len/integer0: None/[null] returnsNone; [false]/[0] retain selected values; one-codepoint string selects itself, >1 is ambiguity, one-member object is integer-key error and truthy scalar has no len. Arbitrary expression JSON has no fake Role record.
  Tests: `identity/role_record_get_test.go:TestPythonCloudGetRolePolymorphicTruthLenAndIntegerZero`.
- Role fixed default finder retains400/403/404 name/domain fallback, missing/duplicate/late proof and GET-only selected override; copied opts/current location and one callback preserve frozen source headers, nil/owner/caller/private guard and physical401 original/native reauth error chain with zero retry callbacks.
  Tests: `identity/role_record_get_test.go:TestPythonCloudGetRoleDefaultFinderFallbackMissingDuplicateAndVersion`, `identity/role_record_get_test.go:TestPythonCloudGetRoleOptsOnceOwnValuesSourceSnapshotAndLocation`, `identity/role_record_get_test.go:TestPythonCloudGetRoleOwnerCancellationPrivateGuardAndPhysical401`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- The named Cloud signatures expose only name_or_id/filters/domain_id; the SDK-owned closed resolver models that reachable subset plus concrete Go Headers/Microversion/Location. Generic named find_user/find_role **query and Resource/adapter cache/session/config lifecycle remain separate unresolved operations; no wholesale Find or supported parity promotion is claimed.
- Concrete immutable JSON descriptor records/options and captured source/context replace Python mutable Resource/dirty/subclass object identity. Source loaded primitive/filter policies reuse existing owned profiles. Strict UTF8/JSON representation/safe routes/headers and bounded pagination/link repairs intentionally differ from Python arbitrary values and malformed-body no-op behavior.
- Direct selected client/factory is SourceNoConnection and defaults location:null. Explicit immutable CloudLocation or root Connection getter bridge supplies current scope, while SDK-R1/C1/S1 whole Resource/Connection/session lifecycle remains tracked outside this named mapping.
- Go wraps concrete invalid-option/ambiguity/native/ResponseError/cancellation causes instead of Python TypeError/KeyError/SDKException class identity or process-global deprecation warnings. Source opaque/Resource JMES function/number/representation behavior is bounded by the documented Go JSON evaluator; this is not a claim of every Python runtime object/function type match.
- Inventory is only actual deprecated search rows; final closed resolver exposes only selected Lookup or an actual error receipt, not every discarded fallback candidate row. Returned record/Value is independent of search observation; expression JSON never receives invented resource/HTTP evidence.
- Role filtered expression returns JSON via Value/Expression with Role=nil; ordinary entities[0] selection is a detached actual RoleRecord. Literalnull and selected [null] are success Value=null, while an error has Value=nil. DeprecatedFilters records Source non-None filter branch without warning output.

### `python:cloud/_identity/IdentityCloudMixin/create_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Create`

- create_role is POST /v3/roles with name and domain_id only when it is given.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityDomainGroupRoleHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/update_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Roles, Projects and Users FindIdentity followed by Update send the two lookups (domain_id on both when given) and then PATCH /<kind>/<id> with only the caller-supplied keys (enabled/description, name, email through Extra); a missing row sends no PATCH, and WithIdentityFindIgnoreMissing(false) returns resource.ErrNotFound like update_project ignore_missing=False.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudFindThenUpdate`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns False when the role is missing; the Go lookup returns a nil row.
- Python commits only attributes that differ from the found Resource and skips the PATCH when nothing changed; Go Update sends every field the caller sets, so an unchanged value is still sent.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- update_role also forwards domain_id to Proxy.update_role; Python sends it only when it differs from the found role, and Go sends it only when the caller adds it through UpdateOpts.Extra.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/delete_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Groups, Roles, Projects and Users FindIdentity followed by Remove(resource.ID(id)) send GET /<kind>/<name> and GET /<kind>?name=<name> (with domain_id on both when WithIdentityFindQuery adds it) and then DELETE /<kind>/<id>; 204 and 404 return nil like the Proxy delete default ignore_missing=True, 409 is returned, and a missing row sends no DELETE.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudFindThenRemove`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns False when the role is missing and re-raises SDKException from the delete.
- Python re-raises SDKException from the delete; Go returns the error (for example 409), and Remove returns nil for 404 like the Proxy delete_role default ignore_missing=True.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/list_role_assignments`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ListAssignments`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListAssignmentsOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListAssignmentsQuery`

- list_role_assignments sends GET /v3/role_assignments with user.id, group.id, role.id, scope.project.id, scope.domain.id, scope.system, effective and scope.OS-INHERIT:inherited_to exactly as the Python filter renaming produces, and decodes role, scope and actor IDs.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityDomainGroupRoleHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python accepts dict or Resource values and reduces them to their id; Go takes ID strings.
- Python encodes effective=True as the string True; ListAssignmentsOpts.Effective encodes true, so the exact Python spelling needs WithListAssignmentsQuery("effective", "True"), and inherited_to has no typed field and needs WithListAssignmentsQuery("scope.OS-INHERIT:inherited_to", ...).
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/grant_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GrantRole`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Assign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Assign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.FindIdentity`

- _get_grant_revoke_params resolves the domain first, then the role without a domain filter, then user, group and project filtered by the domain ID, each strictly; GrantRole sends exactly that GET/list sequence, then HEAD and one PUT.
  Tests: `connectiontest/connection_py_cloud_compute_role_grants_test.go:TestPyCloudComputeGrantRoleProjectUserChecksThenAssigns`.
- An existing assignment returns false and writes nothing; invalid target combinations fail before any request; a missing row stops before the assignment requests; a canceled context sends nothing.
  Tests: `connectiontest/connection_py_cloud_compute_role_grants_test.go:TestPyCloudComputeGrantRoleExistingAssignmentWritesNothing`, `connectiontest/connection_py_cloud_compute_role_grants_test.go:TestPyCloudComputeGrantRoleRejectsBadTargetsAndMissingRows`.
- Non-404 validation failures and failed writes are surfaced rather than treated as absent or successful.
  Tests: `connectiontest/connection_py_cloud_compute_role_grants_test.go:TestPyCloudComputeGrantRoleSurfacesNon404ValidationAndWriteFailures`.

Python/Go differences:

- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding.
- Only HTTP 404 on the HEAD check means the assignment is absent in Go. Python's raw session call returns false for every non-204 status, so a 403 or 500 is also read as absent there and the write continues; Go returns those statuses as errors.
- A failed assignment write is returned as an error in Go; Python ignores the write status and returns True.
- Python ignores wait and timeout; Go sends no wait and treats a successful write as acknowledgement only.
- Python's system argument only selects the /system route and is not sent; Go uses a boolean. Empty strings mean not given in Go, while Python distinguishes None from an empty string.
- Python Proxy.request can serve lookup GETs from a connection response cache; Go has no client-side GET cache.
- Python returns False for an existing/absent assignment and True after a write; Go returns the same boolean plus an error.

### `python:cloud/_identity/IdentityCloudMixin/revoke_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.RevokeRole`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Unassign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Unassign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.FindIdentity`

- RevokeRole performs the same strict lookups, then HEAD and one DELETE for project, domain or system scope, using the OS-INHERIT route for inherited project/domain assignments.
  Tests: `connectiontest/connection_py_cloud_compute_role_grants_test.go:TestPyCloudComputeRevokeRoleSystemInheritedAndAbsent`, `connectiontest/connection_py_cloud_compute_role_grants_test.go:TestPyCloudComputeGrantRoleProjectUserChecksThenAssigns`.
- An absent assignment returns false and sends no DELETE; invalid input and missing rows fail before assignment requests; cancellation sends nothing.
  Tests: `connectiontest/connection_py_cloud_compute_role_grants_test.go:TestPyCloudComputeRevokeRoleSystemInheritedAndAbsent`, `connectiontest/connection_py_cloud_compute_role_grants_test.go:TestPyCloudComputeGrantRoleRejectsBadTargetsAndMissingRows`.

Python/Go differences:

- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding.
- Only HTTP 404 on the HEAD check means the assignment is absent in Go. Python's raw session call returns false for every non-204 status, so a 403 or 500 is also read as absent there and the write continues; Go returns those statuses as errors.
- A failed assignment write is returned as an error in Go; Python ignores the write status and returns True.
- Python ignores wait and timeout; Go sends no wait and treats a successful write as acknowledgement only.
- Python's system argument only selects the /system route and is not sent; Go uses a boolean. Empty strings mean not given in Go, while Python distinguishes None from an empty string.
- Python Proxy.request can serve lookup GETs from a connection response cache; Go has no client-side GET cache.
- Python returns False for an existing/absent assignment and True after a write; Go returns the same boolean plus an error.

### `python:cloud/_identity/IdentityCloudMixin/list_projects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewProjectReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.ProjectReadCloud.ListProjects`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListProjects`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadNameOrID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadLocation`

- Concrete Keystone v3 Project facade expands filters in insertion order then truthy explicit name/domain override corresponding filter values; declared Body filters run in the leaf and actual203 record/header is retained.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudListTruthfulNameDomainOverrideAndLocalBodyFilter`.
- Falsey null/bool/number/underflow/string/list/map filters become empty kwargs. Every truthy string/bool/number/list fails before HTTP because pinned cloud.list_projects uses dict(**filters), despite its JMESPath documentation.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudFalseyFiltersAndTruthyNonMappingRejectBeforeHTTP`.
- Proxy literal lists/strings/objects are consumed without HTTP or invented Project association; nested unconsumed generator remains in Value with nil JSON. Null/noniterable/type-error Proxy values fail before HTTP.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudProxyLiteralAndOpaqueListResultsDoNotInventHTTP`.
- Captured source headers predate callbacks and remain fixed through actual late503; callback executes once, prior physically exposed Inventory remains, actual response proof and custom cancellation survive.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudSourceHeaderSnapshotCallbackCancellationAndLatePageEvidence`.
- Connection uses cached IdentityV3 Projects client, injects configured CurrentLocation before selection and after opts/nil reset, and preserves owned per-call location. Nil/canceled/missing catalog returns real configuration/cancel causes without fake HTTP receipt.
  Tests: `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsShareCachedIdentityAndInjectLocationBeforeSelection`, `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsPerCallLocationSnapshotAndNoNamePushdownForSearch`, `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsPreflightAndMissingEndpointPreserveRealConfigurationCause`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Concrete functional options own defaults and defensive copies. Runtime Resource class/session builders are SDK-owned; safe UTF-8 routes, nonnegative Go pagination caps, same-origin/path/filter continuation and stronger cycle bounds are deliberate Go policies.
- Source-selected literal link precedence, ignored Keystone dictionary links and limit-on-short-page marker fallback are enabled for this new Project profile without changing existing native List or older engine profiles. Valid RFC HTTP Link remains a documented Go interoperability correction of the pinned Requests uri/url failure.
- SDK-R1/C1/S1 remain separately tracked whole-SDK Connection Resource runtime, optional response cache and session/version negotiation limitations. Direct leaf location is null; typed ProjectReadLocation or Connection bridge supplies configured facts. This named read mapping is not whole-SDK parity or live-cloud validation.
- ProjectListResult.Value is []any and retains opaque Source generators; JSON is populated only for wholly representable results. Object-valued Proxy expressions iterate sorted Go keys rather than Python insertion order, an explicit deterministic Go result policy.

### `python:cloud/_identity/IdentityCloudMixin/search_projects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.ProjectReadCloud.SearchProjects`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchProjects`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithProjectReadLocation`

- Search calls list_projects without a server name hint and then applies exact/glob ID/name plus ordered nested loaded-JSON filters. Wire enabled passes server query but fails local Source Resource membership; is_enabled is the owned attribute name.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudSearchGlobNestedLoadedEqualityAndWireAliasFailures`.
- Truthful Source cloud list expansion rejects truthy JMESPath strings/nonmapping filters before HTTP; falsey filters use the ordinary owned list and literal empty Proxy list produces a nonnil empty search inventory.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudFalseyFiltersAndTruthyNonMappingRejectBeforeHTTP`, `identity/project_record_cloud_test.go:TestPythonProjectCloudProxyLiteralAndOpaqueListResultsDoNotInventHTTP`.
- Typed owned location is supplied before local selection, foreign project IDs clear current names, response location remains untouched in Wire, and caller mutation cannot change location options.
  Tests: `identity/project_record_cloud_test.go:TestPythonProjectCloudLocationBeforeLocalSelectionAndOwnedOverrides`.
- Connection cache and CurrentLocation defaults are applied before filtering; per-call location stays owned, Source list-only name option is not pushed from Search, and preflight/catalog failures preserve actual configuration causes.
  Tests: `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsShareCachedIdentityAndInjectLocationBeforeSelection`, `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsPerCallLocationSnapshotAndNoNamePushdownForSearch`, `connectiontest/connection_project_record_lists_test.go:TestConnectionProjectReadsPreflightAndMissingEndpointPreserveRealConfigurationCause`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python sends filters both to the server through list_projects and again to the local filter.
- Concrete functional options own defaults and defensive copies. Runtime Resource class/session builders are SDK-owned; safe UTF-8 routes, nonnegative Go pagination caps, same-origin/path/filter continuation and stronger cycle bounds are deliberate Go policies.
- Source-selected literal link precedence, ignored Keystone dictionary links and limit-on-short-page marker fallback are enabled for this new Project profile without changing existing native List or older engine profiles. Valid RFC HTTP Link remains a documented Go interoperability correction of the pinned Requests uri/url failure.
- SDK-R1/C1/S1 remain separately tracked whole-SDK Connection Resource runtime, optional response cache and session/version negotiation limitations. Direct leaf location is null; typed ProjectReadLocation or Connection bridge supplies configured facts. This named read mapping is not whole-SDK parity or live-cloud validation.
- Search result JSON is separate from actual Projects/Inventory. Source opaque/nonmapping list items fail as concrete Go option errors when local Resource filtering cannot inspect them; no fake receipt or row association is added.

### `python:cloud/_identity/IdentityCloudMixin/get_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetProjectRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewProjectReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.ProjectReadCloud.GetProject`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.ResolveRecord`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Default/None uses a closed domain_id-only Source finder: literal GET, only clean400403404 fallback with name/domain hints, exact ID/name, default nil missing, duplicate error, GET-only per-call MV and actual descriptor/Wire/page/header/status evidence. Caller options/domain/header/location bytes snapshot before sticky API/client/provider/route/version/context/private outer guard; physical faults retain native401 ErrOriginal value and all causes with no fallback/auth/retry/backoff hiding. Project actual list_projects kwargs-expansion precedes _filter_list: truthy nonmapping/string filters fail before HTTP; falsey filters still select deprecated branch. Truthy raw domain overrides only server domain and preserves original local filters. Private wrapper reuses common actual-truthiness-first mapping for scalar literal Proxy rows without invented metadata.
  Tests: `identity/project_record_get_test.go:TestCloudGetProjectNoneVersusPresentFalseySearch`, `identity/project_record_get_test.go:TestCloudGetProjectLateNativeFailureBeforeLocalFilter`, `identity/project_record_get_test.go:TestCloudGetProjectStickyOptionsCancelAndOuterGuard`, `identity/project_record_get_test.go:TestCloudGetProjectPhysicalCloseSourceCancelKeepActualNativeEvidence`, `identity/project_record_get_test.go:TestCloudGetProjectTruthyFilterKwargsExpansionFailsBeforeHTTP`, `identity/project_record_get_test.go:TestCloudGetProjectProxyLiteralFalseyRowsFollowDictFilterBeforeShape`, `identity/project_record_get_test.go:TestCloudGetProjectRawDomainTruthyOverrideKeepsOriginalLocalFilters`, `identity/v3/projects/python_closed_lookup_test.go:TestClosedProjectDomainSeedAndActualSourceEvidence`, `identity/v3/projects/python_closed_lookup_test.go:TestClosedProjectFallbackVersionsExactIDNameAndDefaultMissing`, `identity/v3/projects/python_closed_lookup_test.go:TestClosedProjectOwnedCallbacksRawDomainAndPreflight`, `identity/v3/projects/python_closed_lookup_test.go:TestClosedProjectPhysicalFaultKeepsNative401ValueAndAllCauses`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Generic raw Resource.fetch/header/session kwargs and mutable Resource/Connection/cache/version metadata remain SDK-R1/C1/S1. Full named Proxy find_project/find_group stays unresolved; closed ResolveRecord exposes only identity/domain_id and owned Go Header/MV/location conveniences.
- Go result separates actual Inventory/record from local arbitrary JSON, with nil Value/record for missing and DeprecatedFilters marker instead of Python warning categories. Literal typed identity, safe route/cursor/blank guards and JSON-bounded operands are intentional Go bounds; no object passthrough or fabricated HTTP metadata.
- Source Project truthy string JMES fails at dict(**filters) despite documentation; Group materialized JMES supports arbitrary JSON and Source First shape/errors. Private scalar wrapper is a local common-predicate adaptation and never an HTTP resource.
- Verified Keystone28.0.0 Project member GET allows admin/system/domain reader and own-project user; fallback/list needs admin/system/domain reader. Group member/list allows admin/system reader or target Group domain reader. Deployment overrides and live acceptance are not established by fixtures.

### `python:cloud/_identity/IdentityCloudMixin/update_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Roles, Projects and Users FindIdentity followed by Update send the two lookups (domain_id on both when given) and then PATCH /<kind>/<id> with only the caller-supplied keys (enabled/description, name, email through Extra); a missing row sends no PATCH, and WithIdentityFindIgnoreMissing(false) returns resource.ErrNotFound like update_project ignore_missing=False.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudFindThenUpdate`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python finds with ignore_missing=False and raises NotFoundException for a missing project.
- Python commits only attributes that differ from the found Resource and skips the PATCH when nothing changed; Go Update sends every field the caller sets, so an unchanged value is still sent.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Python maps the enabled argument to is_enabled; Go sets UpdateOpts.Enabled, and other keyword attributes go into typed fields or UpdateOpts.Extra.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/create_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Create`

- create_project(name, domain_id) is POST /v3/projects with name, domain_id, enabled true and description null (sent through CreateOpts.Extra), and extra keyword arguments such as parent_id are added to the same body.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityProjectUserHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python always passes description, so the default body carries description null; Go omits the typed empty Description and needs CreateOpts.Extra{"description": nil} to send null.
- Project allows unknown body attributes, so Python sends any extra keyword; Go sends them through typed CreateOpts fields or Extra.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/delete_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Groups, Roles, Projects and Users FindIdentity followed by Remove(resource.ID(id)) send GET /<kind>/<name> and GET /<kind>?name=<name> (with domain_id on both when WithIdentityFindQuery adds it) and then DELETE /<kind>/<id>; 204 and 404 return nil like the Proxy delete default ignore_missing=True, 409 is returned, and a missing row sends no DELETE.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudFindThenRemove`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns True or False and logs and swallows SDKException from the delete; Go returns nil when the lookup finds nothing (nil row) or the DELETE succeeds or is a 404, and returns other errors such as 409.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/list_users`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewUserReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.ListUsers`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListName`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListLocation`

- list_users() is GET /v3/users and list_users(domain_id=..., name=...) adds exactly those two query keys.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityProjectUserHelpers`.
- Exact decorated Source cloud list accepts only domain_id/name kwargs; ordinary typed/raw null/false/repeated values are passed to owned users list while all other named kwargs fail before marshal/HTTP. No header/filter/JMES/limit Source kwargs are invented.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudListDefaultAndOnlyDeclaredSourceKwargs`, `identity/user_record_cloud_test.go:TestPythonUserCloudListValidKwargsRejectBeforeMarshalerAndHTTP`.
- Cloud list materializes Source rows without member/project lookup, ignores ordinary Keystone dict next, returns owned declared Value/Users and actual Inventory with no fake receipt, and successful empty list publishes nonnil Users/Inventory.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudListDefaultAndOnlyDeclaredSourceKwargs`, `identity/user_record_cloud_test.go:TestPythonUserCloudListOptionSnapshotsHeadersAndCallbacksOwnMaps`.
- Captured selected service/context around copied map/raw/options/header/version callbacks preserves mutable-input snapshots, initializes callback maps, stops after simultaneous option/cancel/source error and avoids fabricated HTTP evidence; explicit User location never treats default project as ownership.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudListOptionSnapshotsHeadersAndCallbacksOwnMaps`, `identity/user_record_cloud_test.go:TestPythonUserCloudSelectedSourceCancellationAndPreflightCannotCreateReceipt`, `identity/user_record_cloud_test.go:TestPythonUserCloudLocationBeforeFilteringIgnoresDefaultProjectAndRemoteForgery`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Go owns immutable JSON descriptor records, fixed source/context/options, detached actual evidence and explicit polymorphic result instead of mutable Python Resource/session/subclass objects. Native User/List/All/membership APIs retain their current contracts.
- Direct leaf/selected v3 factory has no Python Connection: computed location defaults null and immutable explicit CloudLocation is a Go convenience. No default_project_id authorization or automatic client/scope/name inference is introduced. Whole SDK-R1/C1/S1 mutable Resource/cache/session/Connection/version configuration remain tracked separately.
- Go safe UTF8/fixed-origin/collection/filter/conditional-header bounds, nonnegative integer pagination, stronger cycles and valid RFC HTTP Link interoperability intentionally bound broader Source adapters and the pinned Requests uri/url link defect. Runtime constructor controls fail as concrete preflight/receipt errors rather than arbitrary Python object mutations.
- iter.Seq2 can repeat the captured request on repeated iteration rather than one-shot Python generator state. Proxy pure expression still performs SDK option/source preflight; JSON carrier is absent for nested generators. No live cloud/full SDK or shared-engine supported status is claimed.
- Source valid_kwargs domain_id/name restriction is represented explicitly by WithUserListQuery; typed Headers/Microversion/Location are Go transport conveniences, not extra accepted cloud kwargs. The existing native cloud mapping remains available; the new factory materializes owned Source records.

### `python:cloud/_identity/IdentityCloudMixin/search_users`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewUserReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.SearchUsers`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchLocation`

- search_users lists with GET /v3/users carrying name (only for a non-UUID name_or_id) and domain_id before filtering locally.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityProjectUserHelpers`.
- Source UUID hex/braces/prefix/hyphen detection controls only truthy nonUUID server name hint; domain_id is truthy scope, no user/project lookup occurs. Glob name also reaches server literally before exact/glob ID/name local selection.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSearchExactSourceUUIDDetectionAndNameHint`, `identity/user_record_cloud_test.go:TestPythonUserCloudSearchGlobOrderedLoadedFiltersAndDeclaredMembership`, `identity/user_record_cloud_test.go:TestPythonUserCloudEmptyListAndPythonNoneIdentifier`.
- Search completes all list pages then applies ordered nested mapping with Source membership errors/loaded float and bool equality or materialized-row string JMESPath. Falsey filters preserve rows; reached bad-shape/unknown filters fail afterHTTP; empty identifier selection avoids reached nonmapping filter errors.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSearchGlobOrderedLoadedFiltersAndDeclaredMembership`, `identity/user_record_cloud_test.go:TestPythonUserCloudSearchFalseyAndReachedInvalidFilters`.
- Source cloud JMESPath returns arbitrary scalar/null/list JSON separately from actual User association, retains actual observed Inventory only, and type/syntax failures never borrow accepted203 proof. This is materialized input distinct from Proxy opaque generator.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSearchExpressionsMaterializeThenReturnArbitraryJSON`, `identity/user_record_cloud_test.go:TestPythonUserCloudSearchOpaqueLocalErrorsNeverBorrowPhysicalProof`.
- Every page precedes local evaluation, late503 wins over invalid JMES, actual body/header/status remains native rejection and prior Inventory; explicit current User location is applied before local location filtering, ignores default project/server forgery, and invalid location creates noHTTP.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSearchAllPagesBeforeLocalFailureAndLateHTTPWins`, `identity/user_record_cloud_test.go:TestPythonUserCloudLocationBeforeFilteringIgnoresDefaultProjectAndRemoteForgery`.
- Concrete query/filter/header/MV/location copies plus sticky source/context/options preserve custom cancellation and callback cause, stop later callbacks/HTTP, and keep null/missing/empty-list/Python None string identifier behavior.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSelectedSourceCancellationAndPreflightCannotCreateReceipt`, `identity/user_record_cloud_test.go:TestPythonUserCloudEmptyListAndPythonNoneIdentifier`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Go owns immutable JSON descriptor records, fixed source/context/options, detached actual evidence and explicit polymorphic result instead of mutable Python Resource/session/subclass objects. Native User/List/All/membership APIs retain their current contracts.
- Direct leaf/selected v3 factory has no Python Connection: computed location defaults null and immutable explicit CloudLocation is a Go convenience. No default_project_id authorization or automatic client/scope/name inference is introduced. Whole SDK-R1/C1/S1 mutable Resource/cache/session/Connection/version configuration remain tracked separately.
- Go safe UTF8/fixed-origin/collection/filter/conditional-header bounds, nonnegative integer pagination, stronger cycles and valid RFC HTTP Link interoperability intentionally bound broader Source adapters and the pinned Requests uri/url link defect. Runtime constructor controls fail as concrete preflight/receipt errors rather than arbitrary Python object mutations.
- iter.Seq2 can repeat the captured request on repeated iteration rather than one-shot Python generator state. Proxy pure expression still performs SDK option/source preflight; JSON carrier is absent for nested generators. No live cloud/full SDK or shared-engine supported status is claimed.
- Search uses a typed Go name/domain string and raw JSON filters rather than arbitrary Python object coercion. JMESPath exact-decimal/lexical object-key/strict grammar/custom-type differences are existing explicit common-engine policies; shared dependency parity is not promoted by this row.
- Value is arbitrary expression JSON with Users only for ordinary correlated selection; Inventory contains actual detached rows on later failure, whereas Python raises without returning partial rows. The selected v3 factory does not automatically negotiate an identity version or infer Connection facts.

### `python:cloud/_identity/IdentityCloudMixin/get_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetUserRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewUserReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetDomainIDJSON`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetExpression`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.GetUser`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ResolveRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupDomainIDJSON`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Fixed resolver sends raw nullable/false/zero/empty/repeated/object-key domain query and constructor seed, preserves actual response overwrite/descriptor defaults and independently copied view/wire/page/header/status evidence.
  Tests: `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupDomainSeedRawQueryAndDescriptorEvidence`.
- Source100..399 completion and only clean400/403/404 GET fallback retain name/domain hint, exact ID-or-name selection, missing nil, late page failure and immediate second-match ambiguity. Per-call MV applies GET only; fallback keeps selected source version.
  Tests: `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupCleanStatusFallbackAndLiteralNameHint`, `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupMissingDuplicateLatePageAndGetOnlyMicroversion`.
- Closed rawdomain/header/location opts are immutable; nil/context/custom cancellation/source mutation/private guard block work without fake receipt. Malformed accepted representation retains actual response. Existing profiles/old native behavior are untouched.
  Tests: `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupImmutableOptsSourceCancellationAndExplicitLocation`, `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupPrivateGuardAndMalformedAcceptedReceipt`.
- Physical accepted/rejected/native401 read/close/mixed matrix preserves actual body/header/status and all causes, suppresses every Retry/Reauth/Backoff callback and prevents403/404 fallback; clean renewal/retry keeps frozen domain/header/MV with live token.
  Tests: `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupPhysicalFaultStopsFallbackAndNativeCallbacks`, `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupCleanAuthRetryKeepsFrozenRequest`.
- None/literalnull filters use the fixed finder, but every other falsey value triggers deprecated search. Successful search len1 still calls find_user with original identity/domain and returns its actual different record, preserving the pinned missing-return bug and detached search/final proof.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserFiltersNoneVersusAllFalseyPresence`, `identity/user_record_get_test.go:TestPythonCloudGetUserPreservesSuccessfulSearchThenFindBugAndRealReceipts`.
- User search omits falsey domain and supplies non-UUID server name hint; later finder always receives the same domain including None/false/0/empty. UUID normalized names skip list name hint. No arbitrary Source find kwargs are exposed.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserRawDomainTruthinessSearchVersusFixedFinder`.
- User only tests search truth and len before the final finder: one-element object/string/[null]/[false]/[0] still proceeds; falsey returnsNone, >1 ambiguity and truthy scalar len error stop. No entity[0] indexing or expression-to-record evidence is invented.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserPolymorphicSearchUsesLenOnlyNeverIndexZero`.
- Complete search materialization/ordered loaded local predicates preserve missing/duplicate/short-circuit/localerror and late HTTP priority; successful gate followed by missing/duplicate/late final finder keeps actual prior Inventory but no returned User/Value on error.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserSearchStopsOnMissingDuplicateLocalErrorAndLateHTTP`, `identity/user_record_get_test.go:TestPythonCloudGetUserSearchGateThenFinalFinderMissingFallbackAndLateFailure`.
- Options execute once after selected source capture, copy mutable rawdomain/filter/header/current location, stop owner/cancel/private guard callbacks and preserve actual physical403 from the final finder without fallback/retry/reauth.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserOptionsOnceOwnedFilterDomainLocationAndFrozenSource`, `identity/user_record_get_test.go:TestPythonCloudGetUserOptionCancellationOwnerAndPrivateGuardStopWork`, `identity/user_record_get_test.go:TestPythonCloudGetUserPhysicalForbiddenNeverFindFallsThroughOrRetries`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- With the deprecated filters argument Python runs search_users but still returns the find_user result.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- The named Cloud signatures expose only name_or_id/filters/domain_id; the SDK-owned closed resolver models that reachable subset plus concrete Go Headers/Microversion/Location. Generic named find_user/find_role **query and Resource/adapter cache/session/config lifecycle remain separate unresolved operations; no wholesale Find or supported parity promotion is claimed.
- Concrete immutable JSON descriptor records/options and captured source/context replace Python mutable Resource/dirty/subclass object identity. Source loaded primitive/filter policies reuse existing owned profiles. Strict UTF8/JSON representation/safe routes/headers and bounded pagination/link repairs intentionally differ from Python arbitrary values and malformed-body no-op behavior.
- Direct selected client/factory is SourceNoConnection and defaults location:null. Explicit immutable CloudLocation or root Connection getter bridge supplies current scope, while SDK-R1/C1/S1 whole Resource/Connection/session lifecycle remains tracked outside this named mapping.
- Go wraps concrete invalid-option/ambiguity/native/ResponseError/cancellation causes instead of Python TypeError/KeyError/SDKException class identity or process-global deprecation warnings. Source opaque/Resource JMES function/number/representation behavior is bounded by the documented Go JSON evaluator; this is not a claim of every Python runtime object/function type match.
- Inventory is only actual deprecated search rows; final closed resolver exposes only selected Lookup or an actual error receipt, not every discarded fallback candidate row. Returned record/Value is independent of search observation; expression JSON never receives invented resource/HTTP evidence.
- Pinned get_user intentionally discards the single filtered search entity and performs find_user, so a different final resource or final missing/error is preserved. SearchValue/Expression/Inventory describe the gate only; Value/User/Lookup describe the actual final resolution. Process-global deprecation warning is represented by DeprecatedFilters.

### `python:cloud/_identity/IdentityCloudMixin/get_user_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Get`

- get_user_by_id is GET /v3/users/<id>, decodes the user and returns the 404 as an error.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityProjectUserHelpers`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- The deprecated normalize argument only warns in Python and has no Go counterpart.
- Python raises NotFoundException; Go returns the Gophercloud 404 error.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).

### `python:cloud/_identity/IdentityCloudMixin/update_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Roles, Projects and Users FindIdentity followed by Update send the two lookups (domain_id on both when given) and then PATCH /<kind>/<id> with only the caller-supplied keys (enabled/description, name, email through Extra); a missing row sends no PATCH, and WithIdentityFindIgnoreMissing(false) returns resource.ErrNotFound like update_project ignore_missing=False.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudFindThenUpdate`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- The default_project keyword is not a User attribute, so Python drops it from the PATCH.
- Python commits only attributes that differ from the found Resource and skips the PATCH when nothing changed; Go Update sends every field the caller sets, so an unchanged value is still sent.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Python raises SDKException when the user is missing; Go FindIdentity returns a nil row and no PATCH is sent.
- email has no typed UpdateOpts field, so Go sends it through UpdateOpts.Extra; Python drops domain_id=None before the PATCH and sends domain_id only when it differs from the found user.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/create_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.FindIdentity`

- Without default_project, create_user is POST /v3/users with domain_id, name, enabled true, email null (sent through CreateOpts.Extra) and password or description when given.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- With default_project, create_user maps to Projects.FindIdentity (GET /projects/<p>, then GET /projects?name=<p>) followed by POST /users with default_project_id set from the found project, or without that key when the project is missing.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudCreateUserDefaultProject`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python raises SDKException before HTTP when domain_id is missing; Go sends the request without domain_id.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/delete_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Groups, Roles, Projects and Users FindIdentity followed by Remove(resource.ID(id)) send GET /<kind>/<name> and GET /<kind>?name=<name> (with domain_id on both when WithIdentityFindQuery adds it) and then DELETE /<kind>/<id>; 204 and 404 return nil like the Proxy delete default ignore_missing=True, 409 is returned, and a missing row sends no DELETE.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudFindThenRemove`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns True or False and logs and swallows SDKException from the delete; Go returns nil when the lookup finds nothing (nil row) or the DELETE succeeds or is a 404, and returns other errors such as 409.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/add_user_to_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.AddToGroup`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`

- Once IDs are known, add_user_to_group is PUT /v3/groups/<g>/users/<u>.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Users.FindIdentity then Groups.FindIdentity (each GET by name then name list, without domain_id like get_user/get_group) followed by AddToGroup, IsMemberOfGroup or RemoveFromGroup send PUT, HEAD or DELETE /groups/<g>/users/<u>; a missing group stops before the membership request and HEAD 404 reports false.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudUserGroupMembership`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python raises SDKException when the user or group is missing.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/is_user_in_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.IsMemberOfGroup`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`

- Once IDs are known, is_user_in_group is HEAD /v3/groups/<g>/users/<u>, true for 204 and false for 404 like Group.check_user.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Users.FindIdentity then Groups.FindIdentity (each GET by name then name list, without domain_id like get_user/get_group) followed by AddToGroup, IsMemberOfGroup or RemoveFromGroup send PUT, HEAD or DELETE /groups/<g>/users/<u>; a missing group stops before the membership request and HEAD 404 reports false.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudUserGroupMembership`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python raises SDKException when the user or group is missing.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

### `python:cloud/_identity/IdentityCloudMixin/remove_user_from_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.RemoveFromGroup`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`

- Once IDs are known, remove_user_from_group is DELETE /v3/groups/<g>/users/<u>.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Users.FindIdentity then Groups.FindIdentity (each GET by name then name list, without domain_id like get_user/get_group) followed by AddToGroup, IsMemberOfGroup or RemoveFromGroup send PUT, HEAD or DELETE /groups/<g>/users/<u>; a missing group stops before the membership request and HEAD 404 reports false.
  Tests: `identity/v3/python_rejudge_test.go:TestPythonRejudgeCloudUserGroupMembership`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python raises SDKException when the user or group is missing.
- Python Proxy.request can serve the find GET from the connection response cache when caching is enabled; Go has no client-side GET response cache, so every FindIdentity call sends its GET and list requests.
- Python translates responses into seeded openstacksdk Resources (Resource.fetch/_translate_response descriptor coercion and retained unknown attributes); Go decodes the native Gophercloud identity v3 models and requires a safe nonblank ID in every consumed row.
- In the name-list fallback Go follows Keystone links.next while Python Resource.list does not follow the Keystone links dict or use limit/marker continuation without a limit; Keystone returns next null for these lists, so the default requests are the same.
- Names unsafe for a raw member URL (for example containing "/") are looked up by the list request only in Go, while Python sends the direct GET first and falls back after its 404.
- Duplicate name matches raise DuplicateResource in Python and resource.ErrAmbiguous in Go.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
// Python: conn.create_project("demo", domain_id="default")
p, err := svc.Projects.Create(ctx, projects.CreateOpts{
	Name: "demo", DomainID: "default", Enabled: &enabled,
	Extra: map[string]any{"description": nil},
})
```

```go
// Python: conn.delete_service("nova")
found, err := svc.Services.Find(ctx, resource.ID("nova"), resource.WithIgnoreMissing())
if err == nil && found == nil {
	found, err = svc.Services.Find(ctx, resource.Name("nova"), resource.WithIgnoreMissing())
}
if err == nil && found != nil {
	err = svc.Services.Remove(ctx, resource.ID(found.ID))
}
```

```go
// Python: conn.delete_user("alice", domain_id="default")
user, err := svc.Users.FindIdentity(ctx, "alice",
	resource.WithIdentityFindQuery("domain_id", "default"))
if err == nil && user != nil {
	err = svc.Users.Remove(ctx, resource.ID(user.ID))
}
```

## References

- [IdentityCloudMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py)
- [`identity/v3/python_cloud_parity_test.go`](v3/python_cloud_parity_test.go)
- [`identity/v3/python_rejudge_test.go`](v3/python_rejudge_test.go)
- [Python Records](v3/services/python-records.md)
- [Cloud Endpoints Python](cloud-endpoints-python.md)
- [Python Record Reads](v3/projects/python-record-reads.md)
- [Python Records](v3/groups/python-records.md)
- [Python Records](v3/users/python_records.md)
- [Python Records](v3/roles/python-records.md)
- [Python Records](v3/domains/python-records.md)
- [User·Role get](cloud-user-role-get-python.md)
- [Domain get](cloud-domain-get-python.md)
- [Project·Group get](cloud-project-group-get-python.md)
- [Python Parity Users](v3/python-parity-users.md)
- [Python Parity Catalog](v3/python-parity-catalog.md)
- [Finding Identities](../docs/finding-identities.md)
- [`list_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L82)
- [`search_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L123)
- [`get_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L155)
- [Cloud Project Group Get Python](cloud-project-group-get-python.md)
- [`update_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L207)
- [`create_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L235)
- [`delete_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L263)
- [`list_users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L291)
- [`search_users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L305)
- [`get_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L344)
- [Cloud User Role Get Python](cloud-user-role-get-python.md)
- [`get_user_by_id`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L393)
- [`update_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L421)
- [`create_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L442)
- [`delete_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L469)
- [`add_user_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L500)
- [`is_user_in_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L517)
- [`remove_user_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L533)
- [`create_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L551)
- [`update_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L582)
- [`list_services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L600)
- [`search_services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L611)
- [`get_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L640)
- [`delete_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L675)
- [`create_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L699)
- [`update_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L784)
- [`list_endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L797)
- [`search_endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L808)
- [`get_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L841)
- [`delete_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L872)
- [`create_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L895)
- [`update_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L923)
- [`list_endpoint_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L931)
- [`search_endpoint_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L944)
- [`get_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L972)
- [`delete_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L984)
- [`create_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1007)
- [`update_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1031)
- [`delete_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1073)
- [`list_domains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1113)
- [`search_domains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1125)
- [`get_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1161)
- [Cloud Domain Get Python](cloud-domain-get-python.md)
- [`list_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1209)
- [`search_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1222)
- [`get_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1253)
- [`create_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1291)
- [`update_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1324)
- [`delete_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1355)
- [`list_roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1380)
- [`search_roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1391)
- [`get_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1422)
- [`list_role_assignments`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1503)
- [`create_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1577)
- [`update_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1592)
- [`delete_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1614)
- [`grant_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1687)
- [`revoke_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1812)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [ServiceReadCloud](v3/services/python-records.md)
- [EndpointReadCloud](cloud-endpoints-python.md)
- [Keystone28.0.0 Project](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py#L157-L174)
- [Group](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/common/policies/group.py)
- [Groups](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/api/groups.py)
- [User](v3/users/python_records.md)
- [Role](v3/roles/python-records.md)
- [Domain](v3/domains/python-records.md)
