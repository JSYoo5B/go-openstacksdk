# Python Cloud Parity

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_compute/ComputeCloudMixin/add_flavor_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.AddAccess`

- add_flavor_access(flavor_id, project_id) sends POST flavors/{id}/action with {"addTenantAccess": {"tenant": project_id}} through AddAccess.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudFlavorAccessAndSpecHelpers`.

Python/Go differences:

- Python returns None; Go returns the flavor_access rows from the response.
- Python _action sends an empty Accept header and accepts any status below 400; Go uses the Gophercloud default Accept and the action's expected status.
- Python negotiates min(Flavor _max_microversion 2.61, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/remove_flavor_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.RemoveAccess`

- remove_flavor_access(flavor_id, project_id) sends POST flavors/{id}/action with {"removeTenantAccess": {"tenant": project_id}} through RemoveAccess.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudFlavorAccessAndSpecHelpers`.

Python/Go differences:

- Python returns None; Go returns the remaining flavor_access rows from the response.
- Python _action sends an empty Accept header and accepts any status below 400; Go uses the Gophercloud default Accept and the action's expected status.
- Python negotiates min(Flavor _max_microversion 2.61, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/list_flavor_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListAccesses`

- list_flavor_access(flavor_id) sends GET flavors/{id}/os-flavor-access and ListAccesses yields the same flavor_id and tenant_id rows.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudFlavorAccessAndSpecHelpers`.

Python/Go differences:

- Python returns a list of raw dicts; Go yields typed FlavorAccess values through an iterator, so unknown row keys are dropped.
- Python negotiates min(Flavor _max_microversion 2.61, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/set_flavor_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.CreateExtraSpecs`

- set_flavor_specs(flavor_id, extra_specs) sends POST flavors/{id}/os-extra_specs with {"extra_specs": extra_specs} through CreateExtraSpecs.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudFlavorAccessAndSpecHelpers`.

Python/Go differences:

- Python returns None; Go returns the merged extra_specs map from the response.
- Go ExtraSpecsOpts holds string values only, while Python passes any JSON values through.
- Python negotiates min(Flavor _max_microversion 2.61, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/unset_flavor_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.DeleteExtraSpec`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- unset_flavor_specs(flavor_id, keys) sends DELETE flavors/{id}/os-extra_specs/{key} per key; resource.IgnoreMissing turns a 404 into nil like the proxy's ignore_missing=True and a 409 stops the loop with the error.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudFlavorAccessAndSpecHelpers`.

Python/Go differences:

- Python loops over keys inside the helper; Go calls DeleteExtraSpec once per key and wraps each call in resource.IgnoreMissing.
- Python negotiates min(Flavor _max_microversion 2.61, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/create_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Create`

- create_flavor(name, ram, vcpus, disk) with flavorid='auto', ephemeral=0, swap=0, rxtx_factor=1.0 and is_public=True maps to Create with explicit Swap, Ephemeral, RxTxFactor and IsPublic, and an explicit flavorid and description are sent as id and description.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudFlavorCreateNameAndDelete`.

Python/Go differences:

- Python sends "id": null for flavorid='auto' and "description": null when no description is given; Go omits both keys, which Nova treats the same.
- Go encodes rxtx_factor 1.0 as the JSON number 1.
- Python returns a Flavor Resource; Go returns the Gophercloud Flavor without OS-FLV-DISABLED:disabled or original_name.
- Python negotiates min(Flavor _max_microversion 2.61, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/delete_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`

- delete_flavor(name_or_id) sends GET flavors/{name_or_id}, falls back to GET flavors/detail?is_public=None and matches the name through FindIdentity, then Remove sends DELETE flavors/{id} ignoring 404; a missing flavor returns nil before any DELETE.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudFlavorCreateNameAndDelete`.

Python/Go differences:

- Python returns True or False; Go returns the found Flavor or nil from FindIdentity and an error from Remove.
- Python replaces any SDKException with SDKException('Unable to delete flavor ...'); Go returns the original lookup or HTTP error.
- Python negotiates min(Flavor _max_microversion 2.61, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/get_flavor_name`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FindIdentity`

- get_flavor_name(flavor_id) is find_flavor without extra specs: FindIdentity sends GET flavors/{id}, falls back to GET flavors/detail?is_public=None, and yields the flavor name or nil for a missing flavor.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudFlavorCreateNameAndDelete`.

Python/Go differences:

- Python returns the name string or None; Go returns the Flavor, whose Name field the caller reads, or nil.
- Python negotiates min(Flavor _max_microversion 2.61, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/get_flavor_by_ram`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AllFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetFlavorByRAM`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMInclude`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMGetExtra`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMGetExtraValue`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorRAMQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListSourceRecords`

- AllFlavors reproduces the list_flavors(get_extra=True) requests that get_flavor_by_ram starts with.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`.
- SDK eagerly consumes every detailed page and required extra-specs fetch before stable ascending RAM selection; default get_extra is true, inline truthy specs skip fetch, tied RAM preserves original page/row order, minimum/include are not pushed down. A late extra/page failure prevents otherwise valid early selection.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMStableSortAllPagesDefaultEnrichmentAndActualProof`, `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMExtraTruthinessInlineSkipAndLateFailureBeforeSelection`.
- Source RAM descriptor and threshold comparison preserve null failure timing, missing0, signed-string-to0, raw bool, float-to-int binary truncation and exact mixed arbitrary-int/loaded-float ordering. Empty inventory raises completed no-match without prematurely validating unused comparison types.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMThresholdJSONNumbersDescriptorsAndNoMatchOrder`.
- Include is a case-sensitive substring for normal string names, not fnmatch. Falsey JSON skips membership; array names use Python-loaded equality and object names use key membership. Wrong types only fail when reached after RAM comparison; null/noniterable below-threshold names do not fail early.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMIncludeSubstringTruthinessContainersAndShortCircuit`.
- Concrete library-owned options snapshot JSON bytes/DTOs/query-option slices and isolate retained callback pointers. Existing flavor headers/version/extra read options are reused; explicit RAM GetExtra is final override, no filters argument is invented. Nil malformed options/context, selected API mutation and custom cancellation fail before HTTP with native cause preserved.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMOwnsOptionsAndInheritedReadTransportOptions`, `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMPreflightSelectedBindingCancellationAndUnsupportedOptions`.
- Flavor and Inventory records are separately cloned; Source Resource/default/location differs from actual Wire row. FlavorRecord.Envelope is the decoded flat row (unchanged for UTF8 input), page Header/Status203 remain actual, and extra receipt201 keeps whole physical extra response and numeric vendor precision. No-match is ErrNoFlavorByRAM, not fabricated HTTP404; local errors reference consumed row evidence.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMStableSortAllPagesDefaultEnrichmentAndActualProof`, `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMThresholdJSONNumbersDescriptorsAndNoMatchOrder`, `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMExtraTruthinessInlineSkipAndLateFailureBeforeSelection`.
- Existing guarded reader preserves native401 actual failed body, physical Read/Close error chain, custom cancel cause and transient/restored client-source invalidation without returning completed selection. Clean503 retry is exercised with original list route. Existing guard/retry invariants remain; the RAM-only Source status policy is covered by the dedicated contract below.
  Tests: `compute/cloud_flavor_ram_test.go:TestCloudFlavorByRAMPhysicalFaultsNative401AndCleanRetry`.
- RAM-only Source collection and conditional enrichment accept100..399 through ListSourceRecords. Existing AllFlavors/Search/Get/ListRecords/standalone FetchExtraSpecs keep200..399. Actual status/header/body are retained; Requests charset decoding is not implied by this policy.
  Tests: `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMSourceStatusListAndEnrichment`, `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMSourceStatusPreservesExistingReaders`.
- RAM-only Source-status list and conditional extra-specs normalize explicit Latin1 and UTF16/32 BOM or LE/BE bytes before JSON decode. Physical list PageEnvelope and extra Envelope/header/status remain original; rows use decoded UTF8 JSON. Selected/inventory proofs are independently cloned. Decode, later-page and semantic failures preserve actual physical response bytes without overwriting a later HTTP rejection; legacy readers keep their decoder policy. Unknown codec and malformed headers fail explicitly under the finite Go response profile.
  Tests: `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMExplicitCharsetsListAndEnrichment`, `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMCharsetFailuresKeepPhysicalEvidence`, `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMCharsetLaterPageAndSemanticFailuresKeepActualProof`, `compute/cloud_flavor_ram_source_status_test.go:TestCloudFlavorRAMCharsetDoesNotChangeLegacyReaders`.

Python/Go differences:

- Go returns owned FlavorRAMResult with selected Flavor and actual consumed Inventory, instead of only mutable Python Flavor. Resource/default/location, Wire, per-row list Envelope and separate extra-specs whole response evidence remain distinct; local no-match uses errors.Is(ErrNoFlavorByRAM), comparison/membership errors use ErrInvalidOption.
- Public any inputs are SDK-owned JSON-domain primitive/container snapshots, not required application builder interfaces. RawMessage preserves numeric spelling. Python arbitrary classes/operators, NaN/Infinity runtime float values, configurable huge-integer string conversion limits and custom runtime callback timing are not reproduced. Safe Go IDs/finite explicit-charset JSON/header/selected-source policy and existing foundation SDK-R1/SDK-C1/SDK-S1 remain explicit; supported increment0.
- Actual imported pinned Source under Python3.14.8 ran49 cloud→NovaProxy→Flavor/Resource cases, plus2 status199 list/extra cases. Only Adapter Response/connection were substituted. Mock/Go/race/main proof is not actual authenticated backend/role/deployment acceptance.
- Existing AllFlavors default false remains unchanged while this helper owns default true. Root Connection forwarding is not added; conn.Compute(ctx) returns the service. Existing native and generic flavor reader policies are preserved. Existing RAM/status tests plus new explicit-charset integration/root validation are distinct from actual Python Source execution.
- Explicit response profile now supports Latin1 and UTF16/32 BOM or LE/BE on RAM Source-status reads. Unmarked charset sniffing, BOMless generic UTF16/32 native-endian decoding, arbitrary Requests codecs and nonfinite/permissive JSON remain documented Go format differences rather than claims of universal Requests decoding. PageEnvelope is physical whole-page bytes while Envelope is the decoded flat row; extra Envelope remains physical response bytes.

### `python:cloud/_compute/ComputeCloudMixin/create_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Create`

- create_aggregate(name, availability_zone) sends POST os-aggregates with name and the given availability_zone through Create.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudAggregateCreateAndList`.

Python/Go differences:

- Python sends "availability_zone": null when no zone is given; Go omits the key, which Nova treats the same.
- Python returns an Aggregate Resource; Go returns the Gophercloud Aggregate with an integer ID.
- Python negotiates min(Aggregate _max_microversion 2.81, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/list_aggregates`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.All`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`

- list_aggregates() sends GET os-aggregates and list_aggregates(filters={'name': ...}) matches the attribute locally, which All reproduces with resource.WithName.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudAggregateCreateAndList`.

Python/Go differences:

- Python returns a lazy generator of Aggregate Resources; Go All returns a slice and List returns an iterator.
- Python turns filter keys that are Aggregate attributes into local filters; in Go the caller passes them as resource.WithName or resource.WithFilter options.
- Python negotiates min(Aggregate _max_microversion 2.81, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/search_aggregates`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.All`

- All reproduces the unfiltered GET os-aggregates request that search_aggregates starts with.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudAggregateCreateAndList`.

Unresolved boundaries:

- Go has no public SDK helper that applies the cloud _utils._filter_list name_or_id glob match plus dictionary or JMESPath filters to aggregates; the caller would have to filter the listed rows by hand.

### `python:cloud/_compute/ComputeCloudMixin/list_hypervisors`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.List`

- Go List reproduces the default details=True GET os-hypervisors/detail request.
  Tests: `compute/v2/hypervisors/python_parity_test.go:TestPythonHypervisorListAlwaysUsesDetailRoute`.

Unresolved boundaries:

- With filters={'hypervisor_hostname_pattern': ...} below microversion 2.53 Python sends GET os-hypervisors/{pattern}/search; Gophercloud v2.15.0 has no binding for that route and an SDK-owned call would be needed.

### `python:cloud/_compute/ComputeCloudMixin/create_server_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.Create`

- create_server_group(name, policies=[p]) at 2.64 or later sends {"server_group": {"name": name, "policy": p}}, which Create sends with CreateOpts.Policy.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerGroupHelpers`, `compute/v2/servergroups/python_parity_test.go:TestPythonServerGroupCreateGetDeleteMatchProxyRequests`.

Python/Go differences:

- Python rewrites policies to policy at 2.64 or later and policy to policies below it; Go sends the fields the caller sets, so the caller picks Policy or Policies for the client microversion.
- Python negotiates min(ServerGroup _max_microversion 2.64, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/list_server_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.List`

- list_server_groups() sends GET os-server-groups and List yields the same groups.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerGroupHelpers`.

Python/Go differences:

- Python returns a list of ServerGroup Resources; Go yields typed ServerGroup values through an iterator.
- Python negotiates min(ServerGroup _max_microversion 2.64, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/search_server_groups`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.SearchServerGroups`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetServerGroup`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudExpression`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.ListCloudRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.FindCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchServerGroups`

- Search consumes every default-project page before exact/glob name/ID and ordered local mapping/JMES filters, without name/filter/all-projects pushdown. Canonical member_ids is used for builtin dict cloud filters; wire members and unknown attrs are not rewritten to valid keys. Source untyped fields stay raw, rules alone gets dictionary getter cast.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsSearchExactGlobOrderedDictCanonicalAndNoPushdown`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCompleteSearchAndDefaultFindStopsSecondDuplicate`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsListDescriptorsAliasesUnknownStatusAndVersion`.
- Absent/null filters use default find; every nonnull value including false/0/empty dict/list/string enters deprecated complete search. Filtered marker exposes the branch. Primitive JMES results are preserved in Value, not invented as a Resource; filtered get follows loaded truthiness, len, index0, retaining false/zero/empty selected elements and nil only for selected null.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsGetPresenceDefaultFindVersusDeprecatedSearch`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsJMESArbitraryJSONAndFilteredGetIndexSemantics`.
- Default find member GET falls back only for clean400/403/404; exact ID/name equality has no fnmatch conversion. Missing defaults nil. It stops on second exact match before a later HTTP page, while filtered get fully consumes later failures before ambiguity. Observed member and consumed fallback Inventory retain separate actual proof.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsDefaultExactNoGlobMissingEmptyCollectionAndNativeErrors`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCompleteSearchAndDefaultFindStopsSecondDuplicate`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsFindEarlyDuplicateEmptyTrailingAndLateFailure`.
- Actual Source empty string is supported as fixed trailing-slash collection GET with an empty ID seed; it is not collapsed with Source None InvalidRequest. Empty fallback never treats decoded null IDs/names as literal empty strings. Other IDs use owned literal path segments with explicit safe Go constraints.
  Tests: `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsFindEarlyDuplicateEmptyTrailingAndLateFailure`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsEmptyExactFallbackNeverMatchesNullAsEmptyString`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsDiscoveryCeilingSelectedSessionAndLiteralSegments`.
- Selected Nova microversion has precedence; unspecified discovery caps against ServerGroup2.64/min/max using existing shared selection and new strict physical fault opt-in. New member/list Source100..399 leaves old native and FindRecord baselines unchanged. Actual records/default views and Wire/whole Envelope/Header/Status remain owned; CurrentLocation never enters query and foreign project names clear.
  Tests: `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsDiscoveryCeilingSelectedSessionAndLiteralSegments`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsListDescriptorsAliasesUnknownStatusAndVersion`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsOwnsOptionsRecordsHeadersAndCurrentLocation`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCleanRetryReauthStrictStatusAndHeaderCapture`.
- Captured service/client/group binding and option/header/location snapshots precede callbacks. Nil malformed options/context/headers/location and custom cancellation fail preflight. Mutable callback snapshots cannot alter later applied filters/headers or stored response evidence.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsOwnsOptionsRecordsHeadersAndCurrentLocation`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsPreflightSelectedAPIOptionsAndCustomCancellation`.
- Physical read/Close/cancel/restored source faults in search/member/discovery for200/401/404/429/503 preserve actual proof and prevent user retry/reauth/backoff, fallback or missing. Native401 value ErrOriginal/ErrReauth remain available. Clean503 retry and401 token renewal remain guarded. OkCodes expansion cannot establish success or404 absence under the original policy.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsPhysicalFaultsCannotFallbackRetryOrBecomeMissing`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCleanRetryReauthStrictStatusAndHeaderCapture`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsExpandedRejected404AndPhysicalFailureDoNotFallback`.
- New Source opt-in closes finite JSON float loading in local mapping, loaded falsey first selection and exact binary finite operands for JMES comparisons without rounding arbitrary integers. Source two-float9007199254740992.0/9007199254740993.0 comparison is supported; this does not claim full arithmetic, serialization, nonfinite or Resource class equality/function namespace.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsLoadedNumericMappingAndJMESPredicate`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsJMESArbitraryJSONAndFilteredGetIndexSemantics`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`.

Python/Go differences:

- Concrete library-owned Go options/carriers expose Value plus actual Groups/Group/Observed/Inventory instead of only mutable Python Resource/lists. JMES arbitrary output has no fabricated row association. Filtered marks deprecated get-search selection instead of Python warnings.warn side effects.
- Source None name is not representable by the required Go string argument: Source get(None) raisesInvalidRequest without HTTP, while explicit Go empty string is supported as the actual Source trailing collection GET. Search empty is equivalent to Source absent identifier. Safe literal ID/control/traversal/whitespace policy is explicit and never silently replaces an ID.
- Owned header/version/location snapshots, complete UTF8 JSON and JSON object wire order are Go policy. Existing mutation/API caches and native read/find baselines remain intact. Custom runtime objects/mutable Resource/class/adapter/cache/session/current_location foundation differences remain SDK-R1/SDK-C1/SDK-S1 with supported increment0, separately from the two reachable named JMES gaps retained above.
- Nova32 index/show policy is project PROJECT_READER_OR_ADMIN; all_projects index requires ADMIN. Cloud methods keep all_projectsFalse and do not expand privileges. Runtime/mock/race/main proof is not actual authenticated backend, role, ownership or deployment-policy acceptance.
- Actual imported pinned cloud search/get/list→Nova Proxy→Resource list/find/constructor/descriptor→cloudfilter/JMESPath ran under Python3.14.8 in core114 + class/numeric/empty25 + arithmetic8 =147 cases. Only Adapter HTTP/get_endpoint_data/connection were fixture substitutions. Source files and supplemental references are pinned, with no live backend. Go validation covers 15 basic groups; unresolved named class/numeric namespaces prevent completed/full-supported promotion.
- Source None name is not representable by the required Go string argument: Source get(None) raisesInvalidRequest without HTTP, while explicit Go empty string is supported as the actual Source trailing collection GET. Search empty is equivalent to Source absent identifier. Safe literal ID/control/traversal/whitespace policy is explicit and never silently replaces an ID. Literal db0* member GET uses Go single-segment db0%2A encoding versus Source Requests retained star; both default find paths use exact matching, not a glob.

Unresolved boundaries:

- Named JMESPath Source ServerGroup subclass typing/equality remains unresolved. Actual pinned Source type([0]) is object but keys([0])/values([0])/length([0]) raise JMESPathTypeError unknown/ServerGroup; Go JSON map loses that concrete validator. Resource.__eq__ is false against an otherwise identical plain canonical10field object, so Source [?@ == identical_literal].id is [] and filtered get None, whereas structural Go maps can match. Nested rules/metadata dicts are distinct from the Resource class. Actual147-case reference includes these reachable operations; they are not hidden as only generic SDK-R1.
- Named full Python numeric/arithmetic/serialization/nonfinite JMES namespace remains unresolved. Finite binary-loaded comparison and mixed int/float predicate handling are implemented, but arithmetic/serialization retain the existing exact-number engine rather than a complete CPython typed numeric model. Actual Source sum(`[9007199254740992.0,1]`) returns9007199254740992.0; to_string(`1e23`) returns"1e+23". Source literal1e400 search returns inf and filtered get raisesTypeError, while Go loaded JSON rejects nonfinite as unsupported. These named paths stay open, not a claim of full filter parity.

### `python:cloud/_compute/ComputeCloudMixin/get_server_group`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.SearchServerGroups`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetServerGroup`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudExpression`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.ListCloudRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.FindCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetServerGroup`

- Without filters get_server_group(name_or_id) is find_server_group, and FindRecord sends the same GET os-server-groups/{id} with the list fallback.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerGroupHelpers`, `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordUsesGETThenExactIDOrNameAndAllProjects`.
- Search consumes every default-project page before exact/glob name/ID and ordered local mapping/JMES filters, without name/filter/all-projects pushdown. Canonical member_ids is used for builtin dict cloud filters; wire members and unknown attrs are not rewritten to valid keys. Source untyped fields stay raw, rules alone gets dictionary getter cast.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsSearchExactGlobOrderedDictCanonicalAndNoPushdown`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCompleteSearchAndDefaultFindStopsSecondDuplicate`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsListDescriptorsAliasesUnknownStatusAndVersion`.
- Absent/null filters use default find; every nonnull value including false/0/empty dict/list/string enters deprecated complete search. Filtered marker exposes the branch. Primitive JMES results are preserved in Value, not invented as a Resource; filtered get follows loaded truthiness, len, index0, retaining false/zero/empty selected elements and nil only for selected null.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsGetPresenceDefaultFindVersusDeprecatedSearch`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsJMESArbitraryJSONAndFilteredGetIndexSemantics`.
- Default find member GET falls back only for clean400/403/404; exact ID/name equality has no fnmatch conversion. Missing defaults nil. It stops on second exact match before a later HTTP page, while filtered get fully consumes later failures before ambiguity. Observed member and consumed fallback Inventory retain separate actual proof.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsDefaultExactNoGlobMissingEmptyCollectionAndNativeErrors`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCompleteSearchAndDefaultFindStopsSecondDuplicate`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsFindEarlyDuplicateEmptyTrailingAndLateFailure`.
- Actual Source empty string is supported as fixed trailing-slash collection GET with an empty ID seed; it is not collapsed with Source None InvalidRequest. Empty fallback never treats decoded null IDs/names as literal empty strings. Other IDs use owned literal path segments with explicit safe Go constraints.
  Tests: `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsFindEarlyDuplicateEmptyTrailingAndLateFailure`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsEmptyExactFallbackNeverMatchesNullAsEmptyString`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsDiscoveryCeilingSelectedSessionAndLiteralSegments`.
- Selected Nova microversion has precedence; unspecified discovery caps against ServerGroup2.64/min/max using existing shared selection and new strict physical fault opt-in. New member/list Source100..399 leaves old native and FindRecord baselines unchanged. Actual records/default views and Wire/whole Envelope/Header/Status remain owned; CurrentLocation never enters query and foreign project names clear.
  Tests: `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsDiscoveryCeilingSelectedSessionAndLiteralSegments`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsListDescriptorsAliasesUnknownStatusAndVersion`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsOwnsOptionsRecordsHeadersAndCurrentLocation`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCleanRetryReauthStrictStatusAndHeaderCapture`.
- Captured service/client/group binding and option/header/location snapshots precede callbacks. Nil malformed options/context/headers/location and custom cancellation fail preflight. Mutable callback snapshots cannot alter later applied filters/headers or stored response evidence.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsOwnsOptionsRecordsHeadersAndCurrentLocation`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsPreflightSelectedAPIOptionsAndCustomCancellation`.
- Physical read/Close/cancel/restored source faults in search/member/discovery for200/401/404/429/503 preserve actual proof and prevent user retry/reauth/backoff, fallback or missing. Native401 value ErrOriginal/ErrReauth remain available. Clean503 retry and401 token renewal remain guarded. OkCodes expansion cannot establish success or404 absence under the original policy.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsPhysicalFaultsCannotFallbackRetryOrBecomeMissing`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCleanRetryReauthStrictStatusAndHeaderCapture`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsExpandedRejected404AndPhysicalFailureDoNotFallback`.
- New Source opt-in closes finite JSON float loading in local mapping, loaded falsey first selection and exact binary finite operands for JMES comparisons without rounding arbitrary integers. Source two-float9007199254740992.0/9007199254740993.0 comparison is supported; this does not claim full arithmetic, serialization, nonfinite or Resource class equality/function namespace.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsLoadedNumericMappingAndJMESPredicate`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsJMESArbitraryJSONAndFilteredGetIndexSemantics`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`.

Python/Go differences:

- Concrete library-owned Go options/carriers expose Value plus actual Groups/Group/Observed/Inventory instead of only mutable Python Resource/lists. JMES arbitrary output has no fabricated row association. Filtered marks deprecated get-search selection instead of Python warnings.warn side effects.
- Source None name is not representable by the required Go string argument: Source get(None) raisesInvalidRequest without HTTP, while explicit Go empty string is supported as the actual Source trailing collection GET. Search empty is equivalent to Source absent identifier. Safe literal ID/control/traversal/whitespace policy is explicit and never silently replaces an ID.
- Owned header/version/location snapshots, complete UTF8 JSON and JSON object wire order are Go policy. Existing mutation/API caches and native read/find baselines remain intact. Custom runtime objects/mutable Resource/class/adapter/cache/session/current_location foundation differences remain SDK-R1/SDK-C1/SDK-S1 with supported increment0, separately from the two reachable named JMES gaps retained above.
- Nova32 index/show policy is project PROJECT_READER_OR_ADMIN; all_projects index requires ADMIN. Cloud methods keep all_projectsFalse and do not expand privileges. Runtime/mock/race/main proof is not actual authenticated backend, role, ownership or deployment-policy acceptance.
- Actual imported pinned cloud search/get/list→Nova Proxy→Resource list/find/constructor/descriptor→cloudfilter/JMESPath ran under Python3.14.8 in core114 + class/numeric/empty25 + arithmetic8 =147 cases. Only Adapter HTTP/get_endpoint_data/connection were fixture substitutions. Source files and supplemental references are pinned, with no live backend. Go validation covers 15 basic groups; unresolved named class/numeric namespaces prevent completed/full-supported promotion.
- Source None name is not representable by the required Go string argument: Source get(None) raisesInvalidRequest without HTTP, while explicit Go empty string is supported as the actual Source trailing collection GET. Search empty is equivalent to Source absent identifier. Safe literal ID/control/traversal/whitespace policy is explicit and never silently replaces an ID. Literal db0* member GET uses Go single-segment db0%2A encoding versus Source Requests retained star; both default find paths use exact matching, not a glob.

Unresolved boundaries:

- Named JMESPath Source ServerGroup subclass typing/equality remains unresolved. Actual pinned Source type([0]) is object but keys([0])/values([0])/length([0]) raise JMESPathTypeError unknown/ServerGroup; Go JSON map loses that concrete validator. Resource.__eq__ is false against an otherwise identical plain canonical10field object, so Source [?@ == identical_literal].id is [] and filtered get None, whereas structural Go maps can match. Nested rules/metadata dicts are distinct from the Resource class. Actual147-case reference includes these reachable operations; they are not hidden as only generic SDK-R1.
- Named full Python numeric/arithmetic/serialization/nonfinite JMES namespace remains unresolved. Finite binary-loaded comparison and mixed int/float predicate handling are implemented, but arithmetic/serialization retain the existing exact-number engine rather than a complete CPython typed numeric model. Actual Source sum(`[9007199254740992.0,1]`) returns9007199254740992.0; to_string(`1e23`) returns"1e+23". Source literal1e400 search returns inf and filtered get raisesTypeError, while Go loaded JSON rejects nonfinite as unsupported. These named paths stay open, not a claim of full filter parity.

### `python:cloud/_compute/ComputeCloudMixin/delete_server_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.Delete`

- delete_server_group(name_or_id) finds the group with GET os-server-groups/{name_or_id} and the list fallback, then sends a strict DELETE whose 404 is an error; a missing group stops before DELETE.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerGroupHelpers`.

Python/Go differences:

- Python returns True or False; Go returns the found ServerGroupRecord or nil from FindRecord and an error from Delete.
- Python negotiates min(ServerGroup _max_microversion 2.64, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/get_server_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`

- get_server_id(name_or_id) sends GET servers/{name_or_id} then GET servers/detail?name=..., and FindRecord yields the server id or nil for a missing server.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerLookupHelpers`.

Python/Go differences:

- Python get_server(name_or_id, bare=True) is find_server with details=True and all_projects=False; Go servers.API.FindRecord has the same defaults and returns a ServerFindRecord whose Resource.Body holds the declared Server fields as JSON instead of a mutable Server Resource.
- Python returns the ID string or None; Go returns the record or nil and the id is read from Resource.Body.
- Python negotiates min(Server _max_microversion 2.100, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/set_server_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.UpdateMetadata`

- set_server_metadata(name_or_id, metadata) finds the server like get_server(bare=True) and then sends POST servers/{id}/metadata with {"metadata": metadata} through UpdateMetadata.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerLookupHelpers`.

Python/Go differences:

- Python get_server(name_or_id, bare=True) is find_server with details=True and all_projects=False; Go servers.API.FindRecord has the same defaults and returns a ServerFindRecord whose Resource.Body holds the declared Server fields as JSON instead of a mutable Server Resource.
- Python raises SDKException for a missing server; in Go FindRecord returns nil and the caller does not call UpdateMetadata.
- Python returns None; Go returns the merged metadata, and Go MetadataOpts holds string values only.

### `python:cloud/_compute/ComputeCloudMixin/delete_server_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.DeleteMetadatum`

- delete_server_metadata(name_or_id, keys) finds the server like get_server(bare=True) and then sends DELETE servers/{id}/metadata/{key} per key through DeleteMetadatum.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerLookupHelpers`.

Python/Go differences:

- Python get_server(name_or_id, bare=True) is find_server with details=True and all_projects=False; Go servers.API.FindRecord has the same defaults and returns a ServerFindRecord whose Resource.Body holds the declared Server fields as JSON instead of a mutable Server Resource.
- Python raises SDKException for a missing server; in Go FindRecord returns nil and the caller does not call DeleteMetadatum.
- Python loops over keys inside the proxy; Go calls DeleteMetadatum once per key.

### `python:cloud/_compute/ComputeCloudMixin/list_server_security_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.ListByServer`

- list_server_security_groups(server) sends GET servers/{id} and GET servers/{id}/os-security-groups, which Get and ListByServer reproduce.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerLookupHelpers`.

Python/Go differences:

- Python returns [] without HTTP when the secgroup_source config is unset or not nova/neutron; Go has no such config gate.
- Python returns the raw security group dicts; Go yields typed SecurityGroup values.
- Python negotiates min(Server _max_microversion 2.100, server maximum) per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:cloud/_compute/ComputeCloudMixin/create_image_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.CreateImage`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFailureStates`

- create_image_snapshot(name, server, wait=True, **metadata) finds the server, sends createImage with name and metadata, reads the image from Glance and waits for active with a 3600 second budget, failing on error.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudCreateImageSnapshotFindsServerThenImage`, `compute/v2/servers/python_parity_test.go:TestPythonServerCreateImageFetchesAndWaitsForGlanceImage`.

Python/Go differences:

- Python get_server(name_or_id, bare=True) is find_server with details=True and all_projects=False; Go servers.API.FindRecord has the same defaults and returns a ServerFindRecord whose Resource.Body holds the declared Server fields as JSON instead of a mutable Server Resource.
- Python always sends "metadata": {} when no metadata kwargs are given; Go omits an empty metadata map, which Nova treats the same.
- Python wait_for_image keeps polling while the image is not found and raises a timeout exception; Go WaitFor returns a 404 at once and reports the timeout as a context deadline error.
- Python pins 2.45 for createImage when the server supports it; Go uses the client microversion and picks the image ID from the body or the Location header accordingly.

### `python:cloud/_compute/ComputeCloudMixin/get_server`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ExpandServerInterfaces`

- FindRecord reproduces the find_server(details=True) lookup that get_server performs before expansion.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerLookupHelpers`.

Unresolved boundaries:

- With the default bare=False Python passes the server through _expand_server, which calls meta.add_server_interfaces (Neutron/Nova floating IP and port lookups that fill public_v4, public_v6, private_v4 and interface_ip, or meta.get_hostvars_from_server when detailed=True); Go Get/Find/List/Update/Rebuild results are not expanded and compute.Service.ExpandServerInterfaces is a separate call that is not integrated into these helpers, so the returned server differs.
- The deprecated filters argument switches to search_servers and _utils._filter_list; Go has no public SDK helper for that path.

### `python:cloud/_compute/ComputeCloudMixin/get_server_by_id`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ExpandServerInterfaces`

Unresolved boundaries:

- With the default bare=False Python passes the server through _expand_server, which calls meta.add_server_interfaces (Neutron/Nova floating IP and port lookups that fill public_v4, public_v6, private_v4 and interface_ip, or meta.get_hostvars_from_server when detailed=True); Go Get/Find/List/Update/Rebuild results are not expanded and compute.Service.ExpandServerInterfaces is a separate call that is not integrated into these helpers, so the returned server differs.
- Python returns None on NotFoundException; Go Get returns the 404 error and only caller-side status handling could produce None.

### `python:cloud/_compute/ComputeCloudMixin/list_servers`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ExpandServerInterfaces`

Unresolved boundaries:

- With the default bare=False Python passes the server through _expand_server, which calls meta.add_server_interfaces (Neutron/Nova floating IP and port lookups that fill public_v4, public_v6, private_v4 and interface_ip, or meta.get_hostvars_from_server when detailed=True); Go Get/Find/List/Update/Rebuild results are not expanded and compute.Service.ExpandServerInterfaces is a separate call that is not integrated into these helpers, so the returned server differs.
- The delegated proxy servers listing is itself unresolved.

### `python:cloud/_compute/ComputeCloudMixin/search_servers`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.List`

Unresolved boundaries:

- Go has no public SDK helper that applies the cloud _utils._filter_list name_or_id glob match plus dictionary or JMESPath filters to servers; the caller would have to filter the listed rows by hand.
- With the default bare=False Python passes the server through _expand_server, which calls meta.add_server_interfaces (Neutron/Nova floating IP and port lookups that fill public_v4, public_v6, private_v4 and interface_ip, or meta.get_hostvars_from_server when detailed=True); Go Get/Find/List/Update/Rebuild results are not expanded and compute.Service.ExpandServerInterfaces is a separate call that is not integrated into these helpers, so the returned server differs.

### `python:cloud/_compute/ComputeCloudMixin/update_server`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Update`

- The find and PUT parts match the proxy calls update_server delegates to.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerGetUpdateDeleteMatchProxyRequests`.

Unresolved boundaries:

- With the default bare=False Python passes the server through _expand_server, which calls meta.add_server_interfaces (Neutron/Nova floating IP and port lookups that fill public_v4, public_v6, private_v4 and interface_ip, or meta.get_hostvars_from_server when detailed=True); Go Get/Find/List/Update/Rebuild results are not expanded and compute.Service.ExpandServerInterfaces is a separate call that is not integrated into these helpers, so the returned server differs.

### `python:cloud/_compute/ComputeCloudMixin/rebuild_server`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Rebuild`

- The rebuild action matches the proxy rebuild_server request.
  Tests: `compute/v2/servers/python_parity_test.go:TestPythonServerRebuildSendsPassedKeysAndReturnsServer`.

Unresolved boundaries:

- With the default bare=False Python passes the server through _expand_server, which calls meta.add_server_interfaces (Neutron/Nova floating IP and port lookups that fill public_v4, public_v6, private_v4 and interface_ip, or meta.get_hostvars_from_server when detailed=True); Go Get/Find/List/Update/Rebuild results are not expanded and compute.Service.ExpandServerInterfaces is a separate call that is not integrated into these helpers, so the returned server differs.
- With wait=True Python runs compute.wait_for_server for timeout seconds and copies adminPass onto the ACTIVE server; python:compute/v2/wait_for_server is unresolved and Go has no helper that keeps the admin password.

### `python:cloud/_compute/ComputeCloudMixin/delete_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk.WithCloudServerWait`
- `github.com/JSYoo5B/go-openstacksdk.WithCloudServerTimeout`
- `github.com/JSYoo5B/go-openstacksdk.WithCloudServerDeleteIPs`
- `github.com/JSYoo5B/go-openstacksdk.WithCloudServerDeleteIPRetry`
- `github.com/JSYoo5B/go-openstacksdk.WithCloudServerFloatingIPOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk.WithServerAddressPolicy`

- With wait=False and delete_ips=False, FindRecord and Remove reproduce the find_server lookup and the DELETE that ignores 404.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerDeleteAndConsolePartialPaths`.
- DeleteServer returns false without DELETE for a missing server, maps a DELETE 404 to false, never replays a failed DELETE, observes absence with wait, keeps (true, err) after an accepted DELETE whose wait times out and rejects a non-positive timeout before HTTP.
  Tests: `connection_r2_cloud_compute_lifecycle_test.go:TestR2CloudComputeDeleteServerMissingServerSendsNoDelete`, `connection_r2_cloud_compute_lifecycle_test.go:TestR2CloudComputeDeleteServerFoundSendsOneDeleteAndMaps404ToFalse`, `connection_r2_cloud_compute_lifecycle_test.go:TestR2CloudComputeDeleteServerDeleteErrorIsNotReplayed`, `connection_r2_cloud_compute_lifecycle_test.go:TestR2CloudComputeDeleteServerWaitObservesAbsence`, `connection_r2_cloud_compute_lifecycle_test.go:TestR2CloudComputeDeleteServerWaitTimeoutKeepsAcceptedDelete`, `connection_r2_cloud_compute_lifecycle_test.go:TestR2CloudComputeDeleteServerRejectsNonPositiveTimeoutBeforeHTTP`.
- With delete_ips the floating addresses found in the server addresses are looked up and deleted before the server DELETE, a floating IP that is not deleted aborts the server DELETE, and without the option no Neutron call is made.
  Tests: `connection_r2_cloud_compute_lifecycle_test.go:TestR2CloudComputeDeleteServerDeleteIPsRemovesFloatingBeforeServer`, `connection_r2_cloud_compute_lifecycle_test.go:TestR2CloudComputeDeleteServerDeleteIPsFailureAbortsServerDelete`, `connection_r2_cloud_compute_lifecycle_test.go:TestR2CloudComputeDeleteServerWithoutDeleteIPsLeavesFloatingAlone`.
- The _has_floating_ips gate skips floating-IP cleanup without any request when floating_ip_source is none; a lookup raising NotFound continues with the next address; the find_server defaults look up by identity and then name=<name> on the detail list; the Nova floating source lists and deletes through os-floating-ips without Neutron.
  Tests: `connection_cloud_server_delete_gates_test.go:TestCloudDeleteServerFloatingSourceNoneSkipsFloatingCleanup`, `connection_cloud_server_delete_gates_test.go:TestCloudDeleteServerFloatingLookupNotFoundContinues`, `connection_cloud_server_delete_gates_test.go:TestCloudDeleteServerFindServerDefaults`, `connection_cloud_server_delete_gates_test.go:TestCloudDeleteServerNovaFloatingSourceDeletesThroughNova`.

Python/Go differences:

- Wait/delete errors are returned as Go errors; a wait failure after an accepted DELETE returns (true, err).
- The floating-IP gate uses the Connection's floating_ip_source (ServerAddressPolicy or clouds.yaml); per-call floating-IP query options do not change it. Floating lookup and delete semantics are those of GetFloatingIP and DeleteFloatingIP.

### `python:cloud/_compute/ComputeCloudMixin/get_server_console`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConsoleOutput`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetServerConsole`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleLength`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleLengthJSON`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithoutServerConsoleLength`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleServerJSON`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleServerRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithoutServerConsoleServer`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConsoleOutputRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCloudConsoleOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCloudConsoleLength`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCloudConsoleLengthJSON`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithoutCloudConsoleLength`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetServerConsole`

- Existing typed FindRecord and ConsoleOutput preserve their original lookup/action and 400 error contract. The separate GetServerConsole helper below owns the clean-action400 empty-string transformation.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerDeleteAndConsolePartialPaths`.
- Source bare=True lookup uses existing FindRecord defaults(details=True/all_projects=False), member first then detailed local exact fallback, no network/image/flavor expansion; dictionary and Source record seed bypass lookup. Missing/duplicate/lookup physical errors precede the action-only400 catch.
  Tests: `compute/cloud_server_console_test.go:TestCloudServerConsoleBareLookupAndSeedBypass`, `compute/cloud_server_console_test.go:TestCloudServerConsoleNameFallbackUsesDetailsLocalScopeAndAmbiguity`, `compute/cloud_server_console_test.go:TestCloudServerConsoleConnectionLocationSnapshotAndGuard`.
- Nil/explicit null length omits JSON field; zero,false,negative,string,array,dictionary,large integer lengths remain raw. Factories and callback application own snapshots; resets select default lookup/length; nil/cancel/client bindings and invalidUTF8 supplied dictionaries fail before workflowHTTP. Malformed lengthJSON and effective dot/control IDs fail before leaf actionHTTP; a cloud string lookup may already have performed HTTP, preserving lookup-before-action phase order.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordLiteralLengthCaseAndRawOutput`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordOptionsAreOwnedAndReset`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordPreflightAndSourceCallback`, `compute/cloud_server_console_test.go:TestCloudServerConsoleLengthServerSnapshotsRecordAndReset`, `compute/cloud_server_console_test.go:TestCloudServerConsoleNilCancelledAndSourceBindingsPreflight`.
- Raw output membership is case-sensitive and cast is not coercion: null/false/0/list/dict values preserved, missing key is Value null/HasOutput false, explicit null is true; empty root lists/quiet strings return null, output-containing nonmapping/rootnull/scalar or malformed/204 JSON retains actual response error.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordMembershipPresenceAndShape`, `compute/cloud_server_console_test.go:TestCloudServerConsoleCleanAction400OnlyAndRawPresence`.
- Falsey dictionary id maps to Source /servers//action; a dictionary-valued id follows the selected Proxy inner-ID extraction once; absent outerID fails before action, absent innerID uses the falsey route. Arbitrary Server constructor attrs/descriptors remain SDK-R1 foundation gaps. Effective ID trims outer slashes, then bounded dot/control validation/literal escaping; Seed does not fabricate lookup response.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordFalseyAndLiteralIDRoutes`, `compute/cloud_server_console_test.go:TestCloudServerConsoleDictionaryFalseyIDsAndMissingKeys`.
- Action onlyclean400 becomes Value JSON emptystring/Unsupported true with actual400 receipt. Dirty400 read/Close/cancel/sticky restoredsource/nativecallback/expandedOkCodes never suppressed; accepted actionread failure preserves independent lookup/action receipts.
  Tests: `compute/cloud_server_console_test.go:TestCloudServerConsoleDirty400NeverBecomesUnsupported`, `compute/cloud_server_console_test.go:TestCloudServerConsoleCleanRetryStopsAndAcceptedReadKeepsSeparateLookup`.
- The remaining unmarked/unknown-codec Requests JSON-decoding and nonfinite numeric branches stay unsupported with actual response proof. Explicit supported charset normalization is covered by the added contract below. Existing Source68 actualCloud/Proxy/Server/Requests2.34.2/urllib32.8.0/CPython3.14.8 observations are reused, not extended; fixedfind adapter and memoryHTTP post remain distinct from live lookup/auth/retry proof.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordRequestsDecodingAndNonFiniteGapStayExplicit`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`, `connection_cycle8_reads_test.go:TestConnectionCycle8ConsoleUsesCachedComputeAndSuppliedDictionary`.
- Accepted100..399 remains original source policy; selectedversion retained, undiscoveredversion uses Server2.100 ceiling. FixedemptyAccept/version/request JSON/origin and copiedsourceheaders surviveclean retry/reauth; faults at200/400/401/403/404/429/500/777 keep physicalheaders/body/status and block new action hooks. ErrUnable original401 receipt is recovered explicitly without dropping ErrReauth. Cloud options pre-capture private readerheaders for both lookup/action while original selected bindings remain guarded. Console discovery explicitly copies NovaProfile with StopFaultRetryCallbacks=true; physicaldiscovery503 read+Close faults preserve actual native proof without a POST or hooks; global profile baseline remains unchanged.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordOriginalStatusAndInformationalPolicy`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordDiscoveryUsesServerCeilingAndSelectedShortcut`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordPhysicalFaultsRetainActualProofAndStopActionRetry`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordRetryOwnershipAndNativeReauthentication`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordOwnedSourceHeadersAndIndependentReceipt`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordFailedReauthReceiptKeepsOriginalAndCause`, `compute/cloud_server_console_test.go:TestCloudServerConsoleCapturesHeadersBeforeOptionsAndAcrossLookupAction`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordDiscoveryFaultOptInStopsCallbacksBeforeAction`.
- Shared responsejson.Normalize decodes explicit UTF16(BOM/LE/BE), UTF32(BOM/LE/BE), ISO8859-1/latin1 before output parsing. Representative Latin1/UTF16-BOM/UTF32BE accepted202 output values are UTF8 JSON while Envelope/Header remain independent actual wire proof. Unknown/malformed charset and supported-charset malformed JSON preserve original203 ResponseError.Body/Header with distinct Unsupported/InvalidOption causes; shared decoder matrix is root owned.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordExplicitCharsetsKeepWireReceipt`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordCharsetFailuresPreserveActualResponse`.

Python/Go differences:

- Gophercloudv2.15.0 and openstacksdk ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe are pinned. Closed JSON options are SDK-owned concrete factories; callers need no interface builder. Value rawJSON plus HasOutput/Unsupported and separate lookup/seed/action evidence replaces dynamic str|None return.
- SDK-R1/C1/S1 Resource class/constructor/session/cache foundations remain separate shared gaps; general Proxy.get_server_console_output and mutable Server/attrs are not newly mapped. No full cloud get_server expansion is claimed.
- Nil/cancel and invalid unsafe route controls are bounded preflight; effective IDs are escaped literal segments inside the captured Nova origin. Falsey and one-level nested dictionary ID Source lanes remain supported.
- Go integer/rawJSON options retain finite literal values instead of prematurely casting Source annotations. Go raw numbers preserve their original decimal spelling/large integers rather than Python float conversion; nonfinite Python values remain an explicit named gap.
- Synthetic final informational199 status tests establish policy only, because actual net/http may process1xx internally. SDK action ownership guards block retry/auth hooks after physical/context/source faults; reused FindRecord existing lookup hook semantics remain unchanged.
- Source oracle substitutes only fixedfind return/errors and post memoryHTTPResponse; no liveNova/auth/ACL/retry evidence. Python mainASTPASS; root owns Go gates and standalone mainbuilds.
- Cycle11 adds only bounded explicit response charset normalization in the console leaf, reused by existing Service/Connection GetServerConsole. Actual Envelope/Header/ResponseError.Body keep wirebytes while Value comes from normalized JSON. Existing Source68 receipts are unchanged; no new Source matrix, codec registry, guess_json_utf or nonfinite support is claimed. Worker ran gofmt only; root owns Go/race and standalone main validation.

Unresolved boundaries:

- Requests Response.json full codec registry, unknown explicit encodings, charset-free BOM/guess_json_utf selection and text fallback remain unmatched. Explicit UTF16(BOM/LE/BE), UTF32(BOM/LE/BE), ISO8859-1/latin1 now normalize through the SDK shared decoder, including the existing Source ISO8859-1 café observation. Unmarked nonUTF8 or unknown/malformed charset still return ErrUnsupported with actual rawresponse. This narrower reachable remainder continues to prevent complete named mapping. Bare utf-16/utf-32 without BOM native-endian selection remains unsupported.
- Python JSON decoder nonfinite float/NaN/Infinity results cannot be represented by current finite rawJSON result. Source output1e400 becomes float inf; current API returns ErrUnsupported+actualreceipt. Full permissive Python JSON numeric loading remains unresolved.

### `python:cloud/_compute/ComputeCloudMixin/add_server_security_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.AddServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AddServerSecurityGroups`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetSecurityGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk.WithSecurityGroupSource`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SecurityGroupCloud`

- The addSecurityGroup action matches the proxy request once the group name is known.
  Tests: `compute/v2/secgroups/python_parity_test.go:TestPythonSecGroupServerFetchAddRemoveMatchProxyRequests`.
- AddServerSecurityGroups finds the server, resolves every group before any action, sends one addSecurityGroup per group in order using name else ID, and returns false without actions for a missing server/group or empty list.
  Tests: `connection_r2_cloud_compute_secgroups_test.go:TestR2CloudComputeAddServerSecurityGroupsResolvesAllThenSendsInOrder`, `connection_r2_cloud_compute_secgroups_test.go:TestR2CloudComputeSecurityGroupsMissingServerOrGroupSendsNoAction`, `connection_r2_cloud_compute_secgroups_test.go:TestR2CloudComputeSecurityGroupsRejectEmptyIdentitiesBeforeHTTP`, `connection_r2_cloud_compute_secgroups_test.go:TestR2CloudComputeSecurityGroupActionFailureStopsSequenceWithoutReplay`.
- The _has_secgroups gate: a secgroup_source other than neutron or nova in any casing (including null or empty) returns network.ErrUnavailableSecurityGroups before any request; only the exact source 'neutron' with the network service enabled uses Neutron, so 'nova', 'Neutron' or has_network false resolve groups through Nova; clouds.yaml secgroup_source is read and WithSecurityGroupSource overrides it.
  Tests: `connection_cloud_secgroup_source_test.go:TestCloudServerSecurityGroupsUnavailableSourceFailsBeforeHTTP`, `connection_cloud_secgroup_source_test.go:TestCloudServerSecurityGroupsNovaSelection`, `connection_endpoint_discovery_test.go:TestCloudsYAMLServiceFlagsAndSecurityGroupSource`.

Python/Go differences:

- Group inputs are identity strings; pre-resolved dict/Resource groups and a single non-list argument are not accepted.
- The server and groups are resolved with Servers.FindRecord (find_server with details, as get_server(bare=True) does) and GetSecurityGroupRecord; their record representation differences are documented on those operations.

### `python:cloud/_compute/ComputeCloudMixin/remove_server_security_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.RemoveServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.RemoveServerSecurityGroups`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetSecurityGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk.WithSecurityGroupSource`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SecurityGroupCloud`

- The removeSecurityGroup action matches the proxy request once the group name is known.
  Tests: `compute/v2/secgroups/python_parity_test.go:TestPythonSecGroupServerFetchAddRemoveMatchProxyRequests`.
- RemoveServerSecurityGroups shares the lookup phase, continues after a 404 removeSecurityGroup and returns false, stops on other errors and never replays an action.
  Tests: `connection_r2_cloud_compute_secgroups_test.go:TestR2CloudComputeRemoveServerSecurityGroupsContinuesPast404AndReportsFalse`, `connection_r2_cloud_compute_secgroups_test.go:TestR2CloudComputeSecurityGroupsMissingServerOrGroupSendsNoAction`, `connection_r2_cloud_compute_secgroups_test.go:TestR2CloudComputeSecurityGroupActionFailureStopsSequenceWithoutReplay`.
- The _has_secgroups gate: a secgroup_source other than neutron or nova in any casing (including null or empty) returns network.ErrUnavailableSecurityGroups before any request; only the exact source 'neutron' with the network service enabled uses Neutron, so 'nova', 'Neutron' or has_network false resolve groups through Nova; clouds.yaml secgroup_source is read and WithSecurityGroupSource overrides it.
  Tests: `connection_cloud_secgroup_source_test.go:TestCloudServerSecurityGroupsUnavailableSourceFailsBeforeHTTP`, `connection_cloud_secgroup_source_test.go:TestCloudServerSecurityGroupsNovaSelection`, `connection_endpoint_discovery_test.go:TestCloudsYAMLServiceFlagsAndSecurityGroupSource`.

Python/Go differences:

- Group inputs are identity strings; pre-resolved dict/Resource groups are not accepted.
- The server and groups are resolved with Servers.FindRecord (find_server with details, as get_server(bare=True) does) and GetSecurityGroupRecord; their record representation differences are documented on those operations.

### `python:cloud/_compute/ComputeCloudMixin/get_compute_limits`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.API.Fetch`

- Without name_or_id, Fetch sends the same GET limits and returns the absolute limits.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudComputeLimitsCurrentProject`.

Unresolved boundaries:

- Python resolves name_or_id with identity v3 find_project(ignore_missing=False), a GET projects/{name_or_id} followed by a name-filtered list; the Go project scopes take an explicit resource.ID or resource.Name and python:identity/v3/find_project itself is unresolved, so the same lookup requests cannot be produced.

### `python:cloud/_compute/ComputeCloudMixin/get_compute_usage`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.API.SingleTenant`

- Once the project ID is known, SingleTenant sends the os-simple-tenant-usage request that get_usage sends.
  Tests: `compute/v2/usage/python_parity_test.go:TestPythonUsageSingleProjectFirstPageMatchesFetch`.

Unresolved boundaries:

- Python resolves name_or_id with identity v3 find_project(ignore_missing=False), a GET projects/{name_or_id} followed by a name-filtered list; the Go project scopes take an explicit resource.ID or resource.Name and python:identity/v3/find_project itself is unresolved, so the same lookup requests cannot be produced.
- Python parses string start and end values as ISO 8601 and raises SDKException for invalid dates; Go takes time values and has no such parsing helper.

### `python:cloud/_compute/ComputeCloudMixin/get_openstack_vars`

Status: `unresolved`.

Unresolved boundaries:

- Python returns meta.get_hostvars_from_server, which expands addresses, flavor, image, volumes, region, cloud and location into an Ansible hostvars mapping; Go has no equivalent helper.

### `python:cloud/_compute/ComputeCloudMixin/get_server_meta`

Status: `unresolved`.

Unresolved boundaries:

- Python returns server_vars from meta.get_hostvars_from_server plus inventory groups from meta.get_groups_from_server; Go has no equivalent helper.

## References

- [cloud/_compute.py](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py)
- [compute v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [Python Parity](v2/python-parity.md)
- [Python Parity Admin](v2/python-parity-admin.md)
- [compute/v2/python_parity_test.go](v2/python_parity_test.go)
- [Cloud Flavor Ram Python](cloud-flavor-ram-python.md)
- [Cloud Server Group Reads Python](cloud-server-group-reads-python.md)
- [Cloud Server Console Python](cloud-server-console-python.md)
- [Server Addresses](server-addresses.md)
