# Python Parity Admin

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services.WithListQuery`

- Services.List sends GET os-services with no query by default, binary/host from ListOpts and limit/marker through WithListQuery exactly as the Service QueryParameters do, carries the configured microversion header, and decodes the service fields Python exposes.
  Tests: `blockstorage/v3/services/python_parity_test.go:TestPythonServicesListMatchesProxyQuery`.
- Body attributes that Python filters locally (status, zone, ...) are applied by the Go caller over the unfiltered GET os-services rows.
  Tests: `blockstorage/v3/services/python_parity_test.go:TestPythonServicesListLocalFilterIsCallerSide`.

Python/Go differences:

- Python sends only binary, host, limit and marker as server query, filters other Service Body attributes locally and drops unknown keys; Go sends every ListOpts/WithListQuery value to the server and leaves local filtering to the caller.
- Python negotiates min(3.32, server maximum) for the list; Go sends the configured client microversion.
- Python follows services_links if present; Go reads a single page (Cinder does not paginate os-services).
- Go Service has no backend_state field and no name alias for binary, and decodes updated_at only in the zone-less layout; Python returns Service resources with every Body attribute.

### `python:block_storage/v3/backend_pools`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/schedulerstats.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/schedulerstats.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/schedulerstats.WithListOptions`

- SchedulerStats.List with WithListQuery("detail", "True") sends Python's exact GET scheduler-stats/get_pools?detail=True (ListOpts.Detail sends detail=true, the Go default omits detail), adds limit/marker as query, and decodes pool name and typed capabilities.
  Tests: `blockstorage/v3/schedulerstats/python_parity_test.go:TestPythonBackendPoolsDetailQuery`.

Python/Go differences:

- Python always sends detail=True from Pools.base_path; the Go default omits detail and returns name-only pools unless WithListQuery("detail", "True") or ListOpts.Detail (lowercase true) is given.
- Python sends only limit and marker as server query, filters name/capabilities locally and drops tenant_id; Go ListOpts.TenantID sends tenant_id to the server.
- Python keeps the whole capabilities dict; Go Capabilities decodes only declared fields and converts infinite/unknown capacity strings.
- Pools has no _max_microversion so Python uses the session default; Go sends the configured client microversion. Go reads a single page.

### `python:block_storage/v3/extensions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/common/extensions.New`
- `github.com/JSYoo5B/go-openstacksdk/common/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3.Service.RawClient`

- common/extensions built on the Cinder service RawClient sends GET {cinder endpoint}/extensions and decodes alias, name, description, links and updated like the Python Extension resource.
  Tests: `blockstorage/v3/python_parity_test.go:TestPythonCinderExtensionsList`.

Python/Go differences:

- The Cinder service struct has no extensions field; Go callers build common/extensions.New from Service.RawClient().
- Python updated_at maps to the Go Updated string; Go also decodes namespace.
- Extension has no _max_microversion so Python uses the session default; Go sends the configured client microversion and reads a single page.

### `python:block_storage/v3/manage_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.API.ManageExisting`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3.Service.RawClient`

- ManageExisting with every Python argument set posts the same {"volume": {...}} body to manageable_volumes, carries a 3.71 header when the client is set to Python's fixed Volume._max_microversion, and decodes the 202 volume.
  Tests: `blockstorage/v3/manageablevolumes/python_parity_test.go:TestPythonManageVolumeRequest`.
- With only host and ref, Go omits the null name/description/volume_type/availability_zone/metadata and false bootable that Python sends, and an explicit null for a typed key is rejected before HTTP.
  Tests: `blockstorage/v3/manageablevolumes/python_parity_test.go:TestPythonManageVolumeRequest`.

Python/Go differences:

- Python always sends name, description, volume_type, availability_zone and metadata (null when unset) and bootable=false; Go omits empty values, which Cinder reads as the same defaults, and cannot send an explicit null for these typed keys.
- Go Ref is map[string]string, so non-string ref values that Python's dict accepts cannot be sent.
- Python fixes the microversion header at 3.71 and falls back to os-volume-manage when the server lacks 3.8; Go sends the configured client microversion (copy RawClient with Microversion 3.71 to match) and has no legacy path.
- Python returns a Volume resource from the response; Go returns the Gophercloud volumes.Volume whose timestamps decode only in the zone-less layout.

### `python:block_storage/v3/summary`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SummaryRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryAllProjects`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummaryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithSummarySingleFetch`

- Source Proxy summary preserves the initial queryless requires_id=False GET followed by all_tenants=False/True aggregate fetch; stock /volumes/summary is separate from /volumes ListSummary. First fetch only declared untyped seed survives omission; explicit null overwrites. Explicit single fetch improves redundant Source request sequence.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryDefaultTwoFetchesAndAllProjects`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryRetainsFirstFetchDeclaredSeedOnly`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummarySingleFetchAndFetchSpecificVersion`.
- BlockStorageSummary fixed class ceiling is 3.36; discovery is used only without selected default, and selected high versions remain uncapped. Fetch override belongs only to second request; explicit empty override omits version headers. Stock availability is3.12+, actual low-version404 retained rather than invented local support.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummarySingleFetchAndFetchSpecificVersion`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryDiscoveryCeilingAndSelectedVersionNotClamped`.
- Untyped id/name/total_size/total_count/metadata and missing-null declarations own exact arbitrary integer/false/list values. Resource/Wire/whole Envelope/Header/status, terminal fetch Requests and independent Discovery do not alias; accepted nonJSON preserves seed/default, invalid UTF8/object shape preserves actual accepted receipt.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryDefaultTwoFetchesAndAllProjects`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryAcceptedNonJSONAndFlatEnvelopeFallback`.
- Owned JSON-domain kwargs follow Requests iterable expansion and null omission, SDK AllProjects remains authoritative. Deep query/map/raw/pointer option snapshots, nil Header-map composition and option reset are captured without changing original service. Controls and invalid Source/context/MV/header configuration reject before marshaler/discovery/HTTP. Pinned fetch headers kwarg is an ordinary Requests dictionary query and is distinct from copied typed actual HTTP Headers; query null omits and only second fetch carries query.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryInvalidControlsBeforeMarshalerOrHTTP`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryQuerySnapshotsRequestsExpansionAndOptionReset`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryHeadersKwargIsQueryAndTypedHeadersArePhysical`.
- Actual200/203/401/404/429/503 Read,Close and mixed faults retain real body/header/status plus every cause; native401 pointer ErrUnableToReauthenticate preserves ErrOriginal/ErrReauth. Willing Retry,Backoff,Reauth stay zero after physical fault; oneHTTP/oneClose. Source read-boundary mutation is sticky after Close restore, custom cancellation is terminal. Clean401 renewal and503 retry keep selected request; body/version/foreign header/omission mutations retain first rejection and stop.
  Tests: `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryPhysicalFaultsKeepNative401PublicFieldsAndNoHooks`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryCleanReauthenticationAndRetryKeepOwnedRequest`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryReadBoundaryGuardAndCancellationCannotResend`, `blockstorage/v3/volumes/python_summary_contracts_test.go:TestPythonVolumeSummaryRetryCannotInjectBodyOrSelectedHeaders`.

Python/Go differences:

- Go library defaults all_projects tofalse through functional options, while pinned Proxy.summary requires an explicit bool. Default retains Source two fetches; WithSummarySingleFetch(true) explicitly improves redundant first request and drops its seed contribution.
- Standalone leaf projection uses computed locationnull with unknown/location retained only inWire, a documented SDK-R1 boundary; Python Connection-derived dynamic Resource dirty/cache/session/warning behavior remains project-level parity work.
- Go Headers is a copied typed extension. Raw Source controls, unsafe route/session rewriting and duplicate all_tenants/requires_id bindings are rejected before marshaler/HTTP rather than allowing Python kwargs collisions after its firstGET. Exact envelope/header/status/fault proof and native original causes remain.
- Source accepted non-JSON bodies retain seed/default. Go additionally requires UTF8 and nonnull object representations for parsed JSON; receipt-bearing error is exposed for invalidshape rather than Python attribute-method exceptions.

### `python:block_storage/v3/get_quota_class_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.GetQuotaClassRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassGetMicroversion`

- GET default/custom class names project the seven declared quota integer fields with missing/null distinction and preserve actual unknown Wire/body/header/status separately. Selected version remains owned and no version discovery is added.
  Tests: `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassGetSevenDescriptorsAndActualWire`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassGetMissingNullAndCopiedProof`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassOwnedOptionsAndNoVersionDiscovery`.
- Owned options/callback/source/context guards preserve selected routing and do not allow malformed parent or bound controls. Physical read/Close/cancel errors retain actual404 evidence without replay or logical missing suppression.
  Tests: `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassOwnedOptionsAndNoVersionDiscovery`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPreflightIDsAndBoundConstructorControls`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassCallbacksSourceAndCancellationStayTerminal`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPhysicalFaultKeeps404AndAllCauses`.

Python/Go differences:

- Go takes a scalar string class name in a fixed /os-quota-class-sets/{className} route rather than mutable Python Resource input. Only omitted GET name defaults to default; explicit empty names are rejected and other names are sent for server validation. Safe escaped routing, finite UTF-8 JSON and independent declared Resource/Wire/Envelope/header/status snapshots replace arbitrary runtime objects and URI formatter overrides.
- Raw update attributes are not coerced on the wire; only the declared returned Resource view applies Source integer/default/null behavior, including bool remaining bool and non-digit strings becoming0. Unknown Body attributes are dropped from the view while actual Wire retains unknown response JSON. Descriptor projection errors are reported without claiming a successful wire operation or Python runtime object identity.
- New scalar-name update seeds id but never commits dirty id; duplicate attrs id is rejected, and no recognized changes produces no HTTP/Discovery or fabricated receipt. Existing mutable Resource equality/recommit, optional response cache and generic adapter/session/runtime SDK-R1/C1/S1 remain whole-SDK goals. Header/version options are explicit Go conveniences and caller custom transport cache bypass is not claimed.
- Cinder QuotaClassSet has no class maximum; the API retains selected version without adding discovery. Nova class ceiling2.56 is not applied to Cinder, and no automatic Connection location is synthesized.

### `python:block_storage/v3/update_quota_class_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.UpdateQuotaClassRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassUpdateMicroversion`

- PUT transmits raw dirty quota_class_set attrs and explicit null, overlays actual response on class id seed and casts only the returned declared view. No attrs or unknown-only fields yields no HTTP, while duplicate id/constructor controls fail before requests.
  Tests: `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassUpdateRawDirtyAttrsAndSeedProjection`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassUpdateNoopUnknownAndExplicitNull`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPreflightIDsAndBoundConstructorControls`.
- Owned options/callback/source/context guards preserve selected routing and do not allow malformed parent or bound controls. Physical read/Close/cancel errors retain actual404 evidence without replay or logical missing suppression.
  Tests: `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassOwnedOptionsAndNoVersionDiscovery`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPreflightIDsAndBoundConstructorControls`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassCallbacksSourceAndCancellationStayTerminal`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPhysicalFaultKeeps404AndAllCauses`.

Python/Go differences:

- Go takes a scalar string class name in a fixed /os-quota-class-sets/{className} route rather than mutable Python Resource input. Only omitted GET name defaults to default; explicit empty names are rejected and other names are sent for server validation. Safe escaped routing, finite UTF-8 JSON and independent declared Resource/Wire/Envelope/header/status snapshots replace arbitrary runtime objects and URI formatter overrides.
- Raw update attributes are not coerced on the wire; only the declared returned Resource view applies Source integer/default/null behavior, including bool remaining bool and non-digit strings becoming0. Unknown Body attributes are dropped from the view while actual Wire retains unknown response JSON. Descriptor projection errors are reported without claiming a successful wire operation or Python runtime object identity.
- New scalar-name update seeds id but never commits dirty id; duplicate attrs id is rejected, and no recognized changes produces no HTTP/Discovery or fabricated receipt. Existing mutable Resource equality/recommit, optional response cache and generic adapter/session/runtime SDK-R1/C1/S1 remain whole-SDK goals. Header/version options are explicit Go conveniences and caller custom transport cache bypass is not claimed.
- Cinder QuotaClassSet has no class maximum; the API retains selected version without adding discovery. Nova class ceiling2.56 is not applied to Cinder, and no automatic Connection location is synthesized.

### `python:block_storage/v3/resource_filters`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.API.ListResult`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListResource`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListControl`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters.WithRecordListProxyJMESValue`

- ExactResourceFilter declared resource str and filters list descriptor plus inherited untyped id/name; scalar-list wrapping, null defaults, complete literal unknown fields and independent row/page/header/status evidence. Source constructor collision and malformed accepted envelope retain whole page proof.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersDeclaredProjectionOwnsLiteralPageEvidence`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersDefaultsNullableListAndPythonStrings`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersBadEnvelopeAndConstructorCollisionKeepAcceptedPage`.
- Source query mapping has include_pagination_defaultsFalse: resource onlyserverquery, repeated Requests/null expansion, localfilters/id/name via common Cinder MatchLocal, unknownkwargs and callerlimit/marker discarded without invokingmarshalers. Known query JSON and nil headers/callback outputs captured atowned boundaries; raw controls reject beforeanymarshal/HTTP.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersSourceQueryExpansionLocalPredicatesAndUnknownDiscard`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersControlsBeforeMarshalerAndOptionSnapshots`.
- Class ceiling3.33 uses commondiscovery absentselected default, highselected retained and explicitempty override skipsdiscovery/omitsversionheaders onallpages. Low selected3.32 retainsrealstock404 rather thanfabricatedlocalUnsupported.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersDiscoveryCeilingAndExplicitVersionOmission`.
- Lazy rawrow cap/limit hint operates beforelocalfilters, stops beforeunconsumedmalformedrows/next; iterator reuse/break andSinglePage retain defaults. Completefixedcollection pager preservesresourcequery and blocksforeignorigin/cycles; SourceemptyEOF/floatunderflownext falsey behavior is retained; All atomicallydropspartialrowsonfailure.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersLocalCapStopsBeforeInvalidRowsAndNext`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersPaginationPreservesFiltersAndBlocksForeignOrCycle`.
- Source generator-based JMESPath is implemented by ProxyListRecords and common OpaqueIdentity. Separate ListResult explicitly improves array-row evaluation by collection before arbitrary JSON transformation; scalar/string/array/null Value has no invented HTTP proof and Records retain actual receipts. Empty expression is no-op; typed row API routes nonempty expression to polymorphic/eager APIs and eager expression failure clears result after observed pages.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersListResultSupportsArbitraryJMESPathWithoutFakeProof`.
- Actual200/203/401/404/429/503 physicalRead/Close/mixed faults retainbody/header/status/cause; native401pointeroriginal/publicErrReauth remain, zeroRetryBackoffReauthcallbacks andoneHTTP/oneClose. Cleanrenewal/retry keepquery/protocol; read-boundarySource mutation stayssticky afterClose restoration and customcancel terminates. Body/version/foreign/auth/protectedomitretrymutations preservefirstrejection.
  Tests: `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersPhysicalFaults401FieldsAndHooksStayTerminal`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersCleanRenewalAndRetryPreserveQueryAndProtocol`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersReadGuardAndCancellationSurviveCloseRestore`, `blockstorage/v3/resourcefilters/python_contracts_test.go:TestPythonCinderResourceFiltersRetryCannotChangeBodyVersionOrHeaderAbsence`.
- Pinned Proxy JMESPath evaluation sees an opaque unconsumed generator before any version discovery/member HTTP. Projection/type return null, checked array functions fail, literal/equality preserve Source values, @ and nested array/object keep pointer identity and do not serialize as JSON. Falsey or non-string JSON filter values retain the lazy generator. Raw jmespath_filters is consumed privately and captured source remains guarded when generator starts discovery/list.
  Tests: `blockstorage/v3/resourcefilters/python_proxy_contracts_test.go:TestPythonCinderResourceFiltersProxyExpressionsObserveOpaqueGeneratorBeforeHTTP`, `blockstorage/v3/resourcefilters/python_proxy_contracts_test.go:TestPythonCinderResourceFiltersProxyPreservesFalseyNonStringAndNestedGeneratorIdentity`, `blockstorage/v3/resourcefilters/python_proxy_contracts_test.go:TestPythonCinderResourceFiltersProxyRawQueryJMESAndLazyDiscoveryKeepCapturedSource`.
- Common MatchLocalLoaded opt-in uses CPython decimal/exponent binary64 values, arbitrary integers and bool subclass equality through ordered nested Resource dictionary filters. Underflow is numeric zero and falsey, float overflow compares as infinity while adjacent big integer precision is retained, recursive array equality and dictionary subset use the same policy. Existing bindings exact-decimal policy stays unchanged.
  Tests: `blockstorage/v3/resourcefilters/python_proxy_contracts_test.go:TestPythonCinderResourceFiltersLoadedLocalFloatsRetainIntegerPrecisionAndNestedSubsets`.

Python/Go differences:

- Standalone computed locationnull is boundedSDK-R1; literal location andunknownfields retainedinWire, PythonConnection dynamicResource dirty/cache/session remainsprojectparitywork.
- Go ProxyListRecords preserves Source passing an opaque unconsumed generator into JMESPath; ListResult separately improves this by collecting declared JSON rows before evaluation. Python comparison main explicitly selects the eager list/to_dict repair. Source scalar/null/opaque nested pointer behavior is covered without service HTTP; arbitrary eager output retains actual Records separately.
- Ownedpager fixesdict links.next andRequestsHTTP Link uri-key mismatch, enforces sameorigin/fixedcollection/preservedfilters/cycles andonevalidwiremarker. Sourceunsafecontinuation/routecontrols arenotreplayed; malformedcontinuation retainsactualwholepageproof.

## References

- [block_storage v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Python Parity](python-parity.md)
- [Native Admin](native-admin.md)
- [Python Find Cleanup](services/python-find-cleanup.md)
- [Python Records](capabilities/python-records.md)
- [Python Records](manageablevolumes/python-records.md)
- [Python Manage](snapshots/python-manage.md)
- [Python Quota Class](quotasets/python-quota-class.md)
- [volume summary](python-summary.md)
- [resource filters](resource-filters-python.md)
- [`services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3470)
- [Service resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/service.py)
- [Services.FindRecord](services/python-find-cleanup.md)
- [`backend_pools`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1616)
- [Pools](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/stats.py)
- [`extensions`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3632)
- [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/extension.py)
- [`manage_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1282)
- [Volume.manage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py)
- [Capabilities.GetRecord](capabilities/python-records.md)
- [snapshot manage/unmanage](snapshots/python-manage.md)
- [Source record API](quotasets/python-quota-class.md)
