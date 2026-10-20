# Consistency Group Snapshots Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/consistency_group_snapshots`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordListQuery`

- Lazy default/detail vs summary listing. Source6 server query names only; declared Body attrs use shared ordered local predicates, unknown attrs/all_projects/all_tenants skipped without Marshaler. LIST bound row constructor collisions reject actual page while GET unknown cls remains Wire. Exact decimal comparison differs from Source binary64 and is documented.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotListSourceQueriesDetailsFiltersAndRawCap`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotListLocalFloatPolicyAndConstructorCollisions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotBoundClsPresenceAndGetWireBoundary`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotListDefaultAcceptOwnsSourceAndExplicitOption`.
- Shared safe pager retains ID markers independently of public record mutation; advertised marker/offset, initial query defaults/filter/limit, cursor family switches and raw caps; cycle/backwards/origin/collection/filter errors preserve actual page and stop before unsafe request. Source JSON Accept beats captured adapter default; explicit per-call Accept override retained.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotListPagingUsesRetainedScalarMarkerAndScope`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotListAdvertisedOffsetAndCursorScope`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotRawPagingInputPrecedesHTTP`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotListDefaultAcceptOwnsSourceAndExplicitOption`.
- Raw known Source controls reject presence before Marshaler/HTTP, even null/false/zero and malformed raw JSON; ordinary uppercase/unknown remains skipped. Owned typed Headers/Microversion/Details/Paginated/MaxItems are alternatives; dynamic route/conflict/JMES arbitrary result shape stays explicit SDK-S1/SDK-J1, not silently supported.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotListReservedControlsFailBeforeMarshalOrHTTP`.
- Shared pager NextTruthy uses Source JSON binary64 decimal/exponent truthiness (1e-400 false, 1e400 truthy), arbitrary integer and container truth. Falsey next terminates when no separate Link/limit fallback exists; truthy invalid type keeps actual page proof and Find never substitutes an earlier candidate. Local filter exact decimals stay separately documented.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotListAndFindBinary64NextTruthiness`.
- One captured volume service precedes attrs Marshaler and callbacks. JSON opts/headers/seeds/attrs are immutable owned snapshots. Nil header maps are ready for caller callbacks. Guard preserves source/context causes and fixed literal member route; no preflight HTTP receipt is invented.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateSourceCaptureMarshalAndConstructorCollisions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateDeclaredDirtyAttrsAndImmutableOptions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Legacy class inherits max_microversion=None, no automatic discovery or class ceiling; absent selected service version remains absent for every operation. Valid selected/explicit service version and source header aliases stay owned; original client is unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotNoAutomaticDiscoveryAndUncappedSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotClassDefaultsUseNoDiscoveryForAllOperations`.
- Generic100..399 Source successes, actual native400..599 rejection receipts, Read/Close/cancellation/sticky source causes retained. Physical faults never replay, invoke native RetryFunc, ignore404 or become successful projection. Clean retries retain immutable JSON/header/version/route and actual rejection on illegal mutation.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindPhysicalRejectionsCannotFallbackReplayOrIgnore`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindCleanRetryAndOwnedHeaders`.
- Physical401 with willing native retry and authentication hooks terminates after one actual HTTP, zero original callbacks and one close. Native *ErrUnableToReauthenticate pointer retains ErrOriginal401 body/header/method/URL and ErrReauth physical causes; receipt copies only actual ErrOriginal evidence. Clean401 renews live token and resends the identical fixed body/route/version; original public callback pointers remain unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotPhysical401StopsRetryAndReauthWithNativeFields`, `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotClean401RenewsTokenAndRetainsRequest`.

Python/Go differences:

- Raw Source list controls cls/self/session/base_path/list_base_path/microversion/headers/__conflicting_attrs/jmespath_filters/resource_type/details/connection/_synchronized reject presence before Marshaler/HTTP instead of silently becoming unknown. Concrete typed alternatives are provided for supported controls; falsey JMESPath controls also reject. Arbitrary JMES result shape is SDK-J1; dynamic session/class/conflict/route/control remains SDK-S1.
- Body next uses Python JSON binary64 float truthiness for decimal/exponent underflow and overflow, distinct from exact-decimal local predicate. Actual error page is retained for truthy invalid next; no independent HTTP proof is created for falsey termination.
- Mapping is finite named proxy defaults/verbs/class descriptors. Full mutable Python Resource/subclass/session/Adapter/cache/dirty state, arbitrary constructor overload/custom routing and live-cloud/backend acceptance are outside this mapping.
- This legacy ConsistencyGroupSnapshot has no class maximum: inherited max_microversion=None performs no automatic discovery. Valid selected/explicit versions remain uncapped; current stock Cinder27 global max3.71 is a server compatibility fact, not a Source class ceiling or proof that explicit3.80 succeeds on a real cloud. Empty per-call absence is a documented Go extension.
- Go owns UTF-8/control/dot/blank identity validation and path escaping for fixed literal segments. Source untyped6 descriptor defaults/null/false/zero/nested values are preserved; unknown input attrs are discarded, unknown response fields remain Wire/Envelope. Unknown project_id never selects location for this class.
- Record Read/Create/Delete/List Headers/Location/Microversion/Seed configs are concrete owned Go extensions. Source Create constructor microversion and generic controls do not become request Body attrs; use the owned controls rather than claim full constructor/session parity. Constructor collisions are presence-based, including null/false/zero.
- Source named Find accepts identity,ignore_missing,details only. Go JSON Query/headers/location/GET-only microversion/finite paging expose a bounded underlying Resource.find profile: unknown fields only GET, declared seed/local attrs on fallback, special all_projects bool/null GET-excluded and also ignored by this class list mapping. Full Resource.find control overload parity is not claimed.
- Source class list mapping has limit/marker/offset/sort_dir/sort_key/sort, no all_projects/all_tenants. Unknown query is ignored without invoking Marshaler. Stable Go key order and exact decimal local equality differ from Python float rounding; actual9007199254740992 versus filter9007199254740993.0 may differ from Source JSON binary64.
- Shared bounded pager supports standard RFC Link url over pinned Source uri lookup, validates route/origin/filter/limit/cycle and one cursor family with nonnegative uint64 advertised offsets. It does not auto-increment explicit offset. Nonnegative single limit/nonblank single marker, raw caps and single-page controls narrow the supported paging input/search domain.
- Go returns actual raw response receipts for accepted/rejected/partial member and terminal Find failures; Python Delete returns None. Logical missing/duplicates have no fabricated receipt or candidate success. Protected route/framing/headers/body/version and absence are owned across native clean retries; physical faults stop callbacks/replay/fallback/ignore.
- Fixtures reuse public Gophercloud testhelper via internal/testcloud and common reader/REST/filter/pager/identity engines. Stock route/current group backend/policy registration was audited from Cinder27.0.0; actual deployment policy override, driver group capability and runtime acceptance remain separate. No Source spelling workaround from legacy ConsistencyGroup delete applies to these snapshot5 routes.

### `python:block_storage/v3/create_consistency_group_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordCreateMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordCreateLocation`

- Fresh known dirty attrs sent once as cgsnapshot JSON wrapper, supplied ID retained, explicit null/false/zero/nested JSON preserved, nil attrs empty object. Unknown attrs are not marshaled; bound constructor reserved-key presence rejected before HTTP. Source does not prevalidate required consistencygroup_id: actual server400/body/header and native cause retained, no prefetch. JSON framing preflight occurs before POST.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateDeclaredDirtyAttrsAndImmutableOptions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateSourceCaptureMarshalAndConstructorCollisions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotBoundClsPresenceAndGetWireBoundary`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateMissingRequiredFieldKeepsServerRejection`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateJSONFramingPrecedesHTTP`.
- One captured volume service precedes attrs Marshaler and callbacks. JSON opts/headers/seeds/attrs are immutable owned snapshots. Nil header maps are ready for caller callbacks. Guard preserves source/context causes and fixed literal member route; no preflight HTTP receipt is invented.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateSourceCaptureMarshalAndConstructorCollisions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateDeclaredDirtyAttrsAndImmutableOptions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Legacy class inherits max_microversion=None, no automatic discovery or class ceiling; absent selected service version remains absent for every operation. Valid selected/explicit service version and source header aliases stay owned; original client is unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotNoAutomaticDiscoveryAndUncappedSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotClassDefaultsUseNoDiscoveryForAllOperations`.
- Generic100..399 Source successes, actual native400..599 rejection receipts, Read/Close/cancellation/sticky source causes retained. Physical faults never replay, invoke native RetryFunc, ignore404 or become successful projection. Clean retries retain immutable JSON/header/version/route and actual rejection on illegal mutation.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindPhysicalRejectionsCannotFallbackReplayOrIgnore`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindCleanRetryAndOwnedHeaders`.
- Physical401 with willing native retry and authentication hooks terminates after one actual HTTP, zero original callbacks and one close. Native *ErrUnableToReauthenticate pointer retains ErrOriginal401 body/header/method/URL and ErrReauth physical causes; receipt copies only actual ErrOriginal evidence. Clean401 renews live token and resends the identical fixed body/route/version; original public callback pointers remain unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotPhysical401StopsRetryAndReauthWithNativeFields`, `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotClean401RenewsTokenAndRetainsRequest`.

Python/Go differences:

- Raw Source list controls cls/self/session/base_path/list_base_path/microversion/headers/__conflicting_attrs/jmespath_filters/resource_type/details/connection/_synchronized reject presence before Marshaler/HTTP instead of silently becoming unknown. Concrete typed alternatives are provided for supported controls; falsey JMESPath controls also reject. Arbitrary JMES result shape is SDK-J1; dynamic session/class/conflict/route/control remains SDK-S1.
- Body next uses Python JSON binary64 float truthiness for decimal/exponent underflow and overflow, distinct from exact-decimal local predicate. Actual error page is retained for truthy invalid next; no independent HTTP proof is created for falsey termination.
- Mapping is finite named proxy defaults/verbs/class descriptors. Full mutable Python Resource/subclass/session/Adapter/cache/dirty state, arbitrary constructor overload/custom routing and live-cloud/backend acceptance are outside this mapping.
- This legacy ConsistencyGroupSnapshot has no class maximum: inherited max_microversion=None performs no automatic discovery. Valid selected/explicit versions remain uncapped; current stock Cinder27 global max3.71 is a server compatibility fact, not a Source class ceiling or proof that explicit3.80 succeeds on a real cloud. Empty per-call absence is a documented Go extension.
- Go owns UTF-8/control/dot/blank identity validation and path escaping for fixed literal segments. Source untyped6 descriptor defaults/null/false/zero/nested values are preserved; unknown input attrs are discarded, unknown response fields remain Wire/Envelope. Unknown project_id never selects location for this class.
- Record Read/Create/Delete/List Headers/Location/Microversion/Seed configs are concrete owned Go extensions. Source Create constructor microversion and generic controls do not become request Body attrs; use the owned controls rather than claim full constructor/session parity. Constructor collisions are presence-based, including null/false/zero.
- Source named Find accepts identity,ignore_missing,details only. Go JSON Query/headers/location/GET-only microversion/finite paging expose a bounded underlying Resource.find profile: unknown fields only GET, declared seed/local attrs on fallback, special all_projects bool/null GET-excluded and also ignored by this class list mapping. Full Resource.find control overload parity is not claimed.
- Source class list mapping has limit/marker/offset/sort_dir/sort_key/sort, no all_projects/all_tenants. Unknown query is ignored without invoking Marshaler. Stable Go key order and exact decimal local equality differ from Python float rounding; actual9007199254740992 versus filter9007199254740993.0 may differ from Source JSON binary64.
- Shared bounded pager supports standard RFC Link url over pinned Source uri lookup, validates route/origin/filter/limit/cycle and one cursor family with nonnegative uint64 advertised offsets. It does not auto-increment explicit offset. Nonnegative single limit/nonblank single marker, raw caps and single-page controls narrow the supported paging input/search domain.
- Go returns actual raw response receipts for accepted/rejected/partial member and terminal Find failures; Python Delete returns None. Logical missing/duplicates have no fabricated receipt or candidate success. Protected route/framing/headers/body/version and absence are owned across native clean retries; physical faults stop callbacks/replay/fallback/ignore.
- Fixtures reuse public Gophercloud testhelper via internal/testcloud and common reader/REST/filter/pager/identity engines. Stock route/current group backend/policy registration was audited from Cinder27.0.0; actual deployment policy override, driver group capability and runtime acceptance remain separate. No Source spelling workaround from legacy ConsistencyGroup delete applies to these snapshot5 routes.

### `python:block_storage/v3/delete_consistency_group_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordDeleteIgnoreMissing`

- Bodyless DELETE of one escaped literal ID, no prefetch. Default ignores only clean native404 and preserves its actual ignored-missing receipt; strict opts retain404. Opaque generic100..399 ACK retained, no successful metadata GET or projected Resource replaces actual DELETE response. Physical404 remains error and cannot be ignored.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotDeleteDefaultMissingStrictAndOpaqueACK`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`.
- One captured volume service precedes attrs Marshaler and callbacks. JSON opts/headers/seeds/attrs are immutable owned snapshots. Nil header maps are ready for caller callbacks. Guard preserves source/context causes and fixed literal member route; no preflight HTTP receipt is invented.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateSourceCaptureMarshalAndConstructorCollisions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateDeclaredDirtyAttrsAndImmutableOptions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Legacy class inherits max_microversion=None, no automatic discovery or class ceiling; absent selected service version remains absent for every operation. Valid selected/explicit service version and source header aliases stay owned; original client is unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotNoAutomaticDiscoveryAndUncappedSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotClassDefaultsUseNoDiscoveryForAllOperations`.
- Generic100..399 Source successes, actual native400..599 rejection receipts, Read/Close/cancellation/sticky source causes retained. Physical faults never replay, invoke native RetryFunc, ignore404 or become successful projection. Clean retries retain immutable JSON/header/version/route and actual rejection on illegal mutation.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindPhysicalRejectionsCannotFallbackReplayOrIgnore`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindCleanRetryAndOwnedHeaders`.
- Physical401 with willing native retry and authentication hooks terminates after one actual HTTP, zero original callbacks and one close. Native *ErrUnableToReauthenticate pointer retains ErrOriginal401 body/header/method/URL and ErrReauth physical causes; receipt copies only actual ErrOriginal evidence. Clean401 renews live token and resends the identical fixed body/route/version; original public callback pointers remain unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotPhysical401StopsRetryAndReauthWithNativeFields`, `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotClean401RenewsTokenAndRetainsRequest`.

Python/Go differences:

- Raw Source list controls cls/self/session/base_path/list_base_path/microversion/headers/__conflicting_attrs/jmespath_filters/resource_type/details/connection/_synchronized reject presence before Marshaler/HTTP instead of silently becoming unknown. Concrete typed alternatives are provided for supported controls; falsey JMESPath controls also reject. Arbitrary JMES result shape is SDK-J1; dynamic session/class/conflict/route/control remains SDK-S1.
- Body next uses Python JSON binary64 float truthiness for decimal/exponent underflow and overflow, distinct from exact-decimal local predicate. Actual error page is retained for truthy invalid next; no independent HTTP proof is created for falsey termination.
- Mapping is finite named proxy defaults/verbs/class descriptors. Full mutable Python Resource/subclass/session/Adapter/cache/dirty state, arbitrary constructor overload/custom routing and live-cloud/backend acceptance are outside this mapping.
- This legacy ConsistencyGroupSnapshot has no class maximum: inherited max_microversion=None performs no automatic discovery. Valid selected/explicit versions remain uncapped; current stock Cinder27 global max3.71 is a server compatibility fact, not a Source class ceiling or proof that explicit3.80 succeeds on a real cloud. Empty per-call absence is a documented Go extension.
- Go owns UTF-8/control/dot/blank identity validation and path escaping for fixed literal segments. Source untyped6 descriptor defaults/null/false/zero/nested values are preserved; unknown input attrs are discarded, unknown response fields remain Wire/Envelope. Unknown project_id never selects location for this class.
- Record Read/Create/Delete/List Headers/Location/Microversion/Seed configs are concrete owned Go extensions. Source Create constructor microversion and generic controls do not become request Body attrs; use the owned controls rather than claim full constructor/session parity. Constructor collisions are presence-based, including null/false/zero.
- Source named Find accepts identity,ignore_missing,details only. Go JSON Query/headers/location/GET-only microversion/finite paging expose a bounded underlying Resource.find profile: unknown fields only GET, declared seed/local attrs on fallback, special all_projects bool/null GET-excluded and also ignored by this class list mapping. Full Resource.find control overload parity is not claimed.
- Source class list mapping has limit/marker/offset/sort_dir/sort_key/sort, no all_projects/all_tenants. Unknown query is ignored without invoking Marshaler. Stable Go key order and exact decimal local equality differ from Python float rounding; actual9007199254740992 versus filter9007199254740993.0 may differ from Source JSON binary64.
- Shared bounded pager supports standard RFC Link url over pinned Source uri lookup, validates route/origin/filter/limit/cycle and one cursor family with nonnegative uint64 advertised offsets. It does not auto-increment explicit offset. Nonnegative single limit/nonblank single marker, raw caps and single-page controls narrow the supported paging input/search domain.
- Go returns actual raw response receipts for accepted/rejected/partial member and terminal Find failures; Python Delete returns None. Logical missing/duplicates have no fabricated receipt or candidate success. Protected route/framing/headers/body/version and absence are owned across native clean retries; physical faults stop callbacks/replay/fallback/ignore.
- Fixtures reuse public Gophercloud testhelper via internal/testcloud and common reader/REST/filter/pager/identity engines. Stock route/current group backend/policy registration was audited from Cinder27.0.0; actual deployment policy override, driver group capability and runtime acceptance remain separate. No Source spelling workaround from legacy ConsistencyGroup delete applies to these snapshot5 routes.

### `python:block_storage/v3/find_consistency_group_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordFindLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordFindPaginated`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordFindMaxItems`

- Pinned named default ignore_missing=True/details=True, clean member400403404 only fallback, no automatic name server/local query, full exact ID-or-name uniqueness scan. Default nil only after complete successful list; strict logical missing, duplicate and late HTTP/row/link/physical errors never return a candidate success or suppressed member receipt.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindDefaultsCleanFallbackAndNoAutomaticNameFilter`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindDirectSuccessUntypedIDAndSeed`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindCompletePagesDuplicateAndLateFailure`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindDefaultMissingStrictAndListRejection`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindLatePhysicalListFailureKeepsActualPage`.
- GET preparation occurs outside the shared identity fallback engine. GET-only microversion override/empty absence never leaks into fallback selected default. Phase clones keep one original source guard and never re-capture or auto-discover. Bounded underlying Resource.find JSON query/fields/seed/location/paging opts own input and reject reserved/null/false/zero malformed controls before HTTP.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindGetOnlyMicroversionWithoutDiscovery`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindQuerySeedFieldsScopeAndOwnedSnapshots`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindReservedAndMalformedInputPrecedesHTTP`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindCapturedSourceOptionCancellationAndNil`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindPagingControlsOffsetAndRawCap`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindDeclaredLocationAndEvidenceDoNotAlias`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindPreparationCannotBecomeMemberFallback`.
- Shared pager NextTruthy uses Source JSON binary64 decimal/exponent truthiness (1e-400 false, 1e400 truthy), arbitrary integer and container truth. Falsey next terminates when no separate Link/limit fallback exists; truthy invalid type keeps actual page proof and Find never substitutes an earlier candidate. Local filter exact decimals stay separately documented.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotListAndFindBinary64NextTruthiness`.
- One captured volume service precedes attrs Marshaler and callbacks. JSON opts/headers/seeds/attrs are immutable owned snapshots. Nil header maps are ready for caller callbacks. Guard preserves source/context causes and fixed literal member route; no preflight HTTP receipt is invented.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateSourceCaptureMarshalAndConstructorCollisions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateDeclaredDirtyAttrsAndImmutableOptions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Legacy class inherits max_microversion=None, no automatic discovery or class ceiling; absent selected service version remains absent for every operation. Valid selected/explicit service version and source header aliases stay owned; original client is unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotNoAutomaticDiscoveryAndUncappedSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotClassDefaultsUseNoDiscoveryForAllOperations`.
- Generic100..399 Source successes, actual native400..599 rejection receipts, Read/Close/cancellation/sticky source causes retained. Physical faults never replay, invoke native RetryFunc, ignore404 or become successful projection. Clean retries retain immutable JSON/header/version/route and actual rejection on illegal mutation.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindPhysicalRejectionsCannotFallbackReplayOrIgnore`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindCleanRetryAndOwnedHeaders`.
- Physical401 with willing native retry and authentication hooks terminates after one actual HTTP, zero original callbacks and one close. Native *ErrUnableToReauthenticate pointer retains ErrOriginal401 body/header/method/URL and ErrReauth physical causes; receipt copies only actual ErrOriginal evidence. Clean401 renews live token and resends the identical fixed body/route/version; original public callback pointers remain unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotPhysical401StopsRetryAndReauthWithNativeFields`, `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotClean401RenewsTokenAndRetainsRequest`.

Python/Go differences:

- Raw Source list controls cls/self/session/base_path/list_base_path/microversion/headers/__conflicting_attrs/jmespath_filters/resource_type/details/connection/_synchronized reject presence before Marshaler/HTTP instead of silently becoming unknown. Concrete typed alternatives are provided for supported controls; falsey JMESPath controls also reject. Arbitrary JMES result shape is SDK-J1; dynamic session/class/conflict/route/control remains SDK-S1.
- Body next uses Python JSON binary64 float truthiness for decimal/exponent underflow and overflow, distinct from exact-decimal local predicate. Actual error page is retained for truthy invalid next; no independent HTTP proof is created for falsey termination.
- Mapping is finite named proxy defaults/verbs/class descriptors. Full mutable Python Resource/subclass/session/Adapter/cache/dirty state, arbitrary constructor overload/custom routing and live-cloud/backend acceptance are outside this mapping.
- This legacy ConsistencyGroupSnapshot has no class maximum: inherited max_microversion=None performs no automatic discovery. Valid selected/explicit versions remain uncapped; current stock Cinder27 global max3.71 is a server compatibility fact, not a Source class ceiling or proof that explicit3.80 succeeds on a real cloud. Empty per-call absence is a documented Go extension.
- Go owns UTF-8/control/dot/blank identity validation and path escaping for fixed literal segments. Source untyped6 descriptor defaults/null/false/zero/nested values are preserved; unknown input attrs are discarded, unknown response fields remain Wire/Envelope. Unknown project_id never selects location for this class.
- Record Read/Create/Delete/List Headers/Location/Microversion/Seed configs are concrete owned Go extensions. Source Create constructor microversion and generic controls do not become request Body attrs; use the owned controls rather than claim full constructor/session parity. Constructor collisions are presence-based, including null/false/zero.
- Source named Find accepts identity,ignore_missing,details only. Go JSON Query/headers/location/GET-only microversion/finite paging expose a bounded underlying Resource.find profile: unknown fields only GET, declared seed/local attrs on fallback, special all_projects bool/null GET-excluded and also ignored by this class list mapping. Full Resource.find control overload parity is not claimed.
- Source class list mapping has limit/marker/offset/sort_dir/sort_key/sort, no all_projects/all_tenants. Unknown query is ignored without invoking Marshaler. Stable Go key order and exact decimal local equality differ from Python float rounding; actual9007199254740992 versus filter9007199254740993.0 may differ from Source JSON binary64.
- Shared bounded pager supports standard RFC Link url over pinned Source uri lookup, validates route/origin/filter/limit/cycle and one cursor family with nonnegative uint64 advertised offsets. It does not auto-increment explicit offset. Nonnegative single limit/nonblank single marker, raw caps and single-page controls narrow the supported paging input/search domain.
- Go returns actual raw response receipts for accepted/rejected/partial member and terminal Find failures; Python Delete returns None. Logical missing/duplicates have no fabricated receipt or candidate success. Protected route/framing/headers/body/version and absence are owned across native clean retries; physical faults stop callbacks/replay/fallback/ignore.
- Fixtures reuse public Gophercloud testhelper via internal/testcloud and common reader/REST/filter/pager/identity engines. Stock route/current group backend/policy registration was audited from Cinder27.0.0; actual deployment policy override, driver group capability and runtime acceptance remain separate. No Source spelling workaround from legacy ConsistencyGroup delete applies to these snapshot5 routes.

### `python:block_storage/v3/get_consistency_group_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordReadLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.WithRecordReadSeed`

- One escaped literal member GET, no query or prefetch. Exactly6 untyped Source Body fields plus recorded-auth/owned location, passive response ID/null/unknown Wire, independent raw/declared/envelope/header carriers. Missing fields are null; empty/malformed JSON tolerates constructor seed, parseable invalid root/wrapper retains actual failure proof.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotGetOwnsUntypedDescriptorsEnvelopeAndLocation`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotGetOpaqueSeedNullAndInvalidShapeProof`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotBoundClsPresenceAndGetWireBoundary`.
- One captured volume service precedes attrs Marshaler and callbacks. JSON opts/headers/seeds/attrs are immutable owned snapshots. Nil header maps are ready for caller callbacks. Guard preserves source/context causes and fixed literal member route; no preflight HTTP receipt is invented.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateSourceCaptureMarshalAndConstructorCollisions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotCreateDeclaredDirtyAttrsAndImmutableOptions`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotOptionsPreflightOwnershipAndNilAPI`.
- Legacy class inherits max_microversion=None, no automatic discovery or class ceiling; absent selected service version remains absent for every operation. Valid selected/explicit service version and source header aliases stay owned; original client is unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotNoAutomaticDiscoveryAndUncappedSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotClassDefaultsUseNoDiscoveryForAllOperations`.
- Generic100..399 Source successes, actual native400..599 rejection receipts, Read/Close/cancellation/sticky source causes retained. Physical faults never replay, invoke native RetryFunc, ignore404 or become successful projection. Clean retries retain immutable JSON/header/version/route and actual rejection on illegal mutation.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess`, `blockstorage/v3/consistencygroupsnapshots/python_records_test.go:TestPythonConsistencyGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindPhysicalRejectionsCannotFallbackReplayOrIgnore`, `blockstorage/v3/consistencygroupsnapshots/python_find_test.go:TestPythonConsistencyGroupSnapshotFindCleanRetryAndOwnedHeaders`.
- Physical401 with willing native retry and authentication hooks terminates after one actual HTTP, zero original callbacks and one close. Native *ErrUnableToReauthenticate pointer retains ErrOriginal401 body/header/method/URL and ErrReauth physical causes; receipt copies only actual ErrOriginal evidence. Clean401 renews live token and resends the identical fixed body/route/version; original public callback pointers remain unchanged.
  Tests: `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotPhysical401StopsRetryAndReauthWithNativeFields`, `blockstorage/v3/consistencygroupsnapshots/python_boundary_contracts_test.go:TestPythonConsistencyGroupSnapshotClean401RenewsTokenAndRetainsRequest`.

Python/Go differences:

- Raw Source list controls cls/self/session/base_path/list_base_path/microversion/headers/__conflicting_attrs/jmespath_filters/resource_type/details/connection/_synchronized reject presence before Marshaler/HTTP instead of silently becoming unknown. Concrete typed alternatives are provided for supported controls; falsey JMESPath controls also reject. Arbitrary JMES result shape is SDK-J1; dynamic session/class/conflict/route/control remains SDK-S1.
- Body next uses Python JSON binary64 float truthiness for decimal/exponent underflow and overflow, distinct from exact-decimal local predicate. Actual error page is retained for truthy invalid next; no independent HTTP proof is created for falsey termination.
- Mapping is finite named proxy defaults/verbs/class descriptors. Full mutable Python Resource/subclass/session/Adapter/cache/dirty state, arbitrary constructor overload/custom routing and live-cloud/backend acceptance are outside this mapping.
- This legacy ConsistencyGroupSnapshot has no class maximum: inherited max_microversion=None performs no automatic discovery. Valid selected/explicit versions remain uncapped; current stock Cinder27 global max3.71 is a server compatibility fact, not a Source class ceiling or proof that explicit3.80 succeeds on a real cloud. Empty per-call absence is a documented Go extension.
- Go owns UTF-8/control/dot/blank identity validation and path escaping for fixed literal segments. Source untyped6 descriptor defaults/null/false/zero/nested values are preserved; unknown input attrs are discarded, unknown response fields remain Wire/Envelope. Unknown project_id never selects location for this class.
- Record Read/Create/Delete/List Headers/Location/Microversion/Seed configs are concrete owned Go extensions. Source Create constructor microversion and generic controls do not become request Body attrs; use the owned controls rather than claim full constructor/session parity. Constructor collisions are presence-based, including null/false/zero.
- Source named Find accepts identity,ignore_missing,details only. Go JSON Query/headers/location/GET-only microversion/finite paging expose a bounded underlying Resource.find profile: unknown fields only GET, declared seed/local attrs on fallback, special all_projects bool/null GET-excluded and also ignored by this class list mapping. Full Resource.find control overload parity is not claimed.
- Source class list mapping has limit/marker/offset/sort_dir/sort_key/sort, no all_projects/all_tenants. Unknown query is ignored without invoking Marshaler. Stable Go key order and exact decimal local equality differ from Python float rounding; actual9007199254740992 versus filter9007199254740993.0 may differ from Source JSON binary64.
- Shared bounded pager supports standard RFC Link url over pinned Source uri lookup, validates route/origin/filter/limit/cycle and one cursor family with nonnegative uint64 advertised offsets. It does not auto-increment explicit offset. Nonnegative single limit/nonblank single marker, raw caps and single-page controls narrow the supported paging input/search domain.
- Go returns actual raw response receipts for accepted/rejected/partial member and terminal Find failures; Python Delete returns None. Logical missing/duplicates have no fabricated receipt or candidate success. Protected route/framing/headers/body/version and absence are owned across native clean retries; physical faults stop callbacks/replay/fallback/ignore.
- Fixtures reuse public Gophercloud testhelper via internal/testcloud and common reader/REST/filter/pager/identity engines. Stock route/current group backend/policy registration was audited from Cinder27.0.0; actual deployment policy override, driver group capability and runtime acceptance remain separate. No Source spelling workaround from legacy ConsistencyGroup delete applies to these snapshot5 routes.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("operation", choices=["get", "find", "list", "create", "delete"])
parser.add_argument("--identity")
parser.add_argument("--group")
parser.add_argument("--name")
parser.add_argument("--attrs", default="{}")
parser.add_argument("--query", default="{}")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--summary", action="store_true")
args = parser.parse_args()
conn = openstack.connect()
if args.operation in {"get", "find", "delete"} and args.identity is None:
    parser.error("--identity is required")
if args.operation == "get":
    value = conn.block_storage.get_consistency_group_snapshot(args.identity)
    print(value.to_dict())
elif args.operation == "find":
    value = conn.block_storage.find_consistency_group_snapshot(
        args.identity, ignore_missing=not args.strict, details=not args.summary
    )
    print(None if value is None else value.to_dict())
elif args.operation == "list":
    for value in conn.block_storage.consistency_group_snapshots(
        details=not args.summary, **json.loads(args.query)
    ):
        print(value.to_dict())
elif args.operation == "create":
    attrs = json.loads(args.attrs)
    if args.group is not None:
        attrs["consistencygroup_id"] = args.group
    if args.name is not None:
        attrs["name"] = args.name
    print(conn.block_storage.create_consistency_group_snapshot(**attrs).to_dict())
else:
    conn.block_storage.delete_consistency_group_snapshot(
        args.identity, ignore_missing=not args.strict
    )
    print("delete returned")
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
)

func main() {
    operation := flag.String("operation", "list", "get, find, list, create or delete")
    identity := flag.String("identity", "", "literal ID, or exact ID/name for find")
    group := flag.String("group", "", "source consistency group ID for create")
    name := flag.String("name", "", "create name, including an explicit empty string")
    attrsJSON := flag.String("attrs", "{}", "create JSON object")
    queryJSON := flag.String("query", "{}", "list JSON query object")
    strict := flag.Bool("strict", false, "report missing find/delete as an error")
    summary := flag.Bool("summary", false, "list summary, also on find fallback")
    flag.Parse()
    supplied := make(map[string]bool)
    flag.Visit(func(f *flag.Flag) { supplied[f.Name] = true })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := consistencygroupsnapshots.New(cinder.RawClient())
    show := func(value *consistencygroupsnapshots.ConsistencyGroupSnapshotRecord, err error) {
        if value != nil {
            fmt.Printf("actual_http=%d actual_body=%q\n", value.StatusCode, value.Envelope)
        }
        if err != nil { log.Fatal(err) }
        if value == nil { fmt.Println("no matching consistency group snapshot"); return }
        raw, err := json.MarshalIndent(value.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(raw))
    }
    switch *operation {
    case "get":
        show(api.GetRecord(ctx, *identity))
    case "find":
        show(api.FindRecord(ctx, *identity,
            consistencygroupsnapshots.WithRecordFindIgnoreMissing(!*strict),
            consistencygroupsnapshots.WithRecordFindDetails(!*summary)))
    case "list":
        var query map[string]json.RawMessage
        if err := json.Unmarshal([]byte(*queryJSON), &query); err != nil { log.Fatal(err) }
        for value, err := range api.ListRecords(ctx,
            consistencygroupsnapshots.WithRecordListOpts(consistencygroupsnapshots.RecordListOpts{Query: query}),
            consistencygroupsnapshots.WithRecordListDetails(!*summary)) {
            show(value, err)
        }
    case "create":
        var attrs map[string]any
        decoder := json.NewDecoder(strings.NewReader(*attrsJSON))
        decoder.UseNumber()
        if err := decoder.Decode(&attrs); err != nil { log.Fatal(err) }
        if attrs == nil { attrs = make(map[string]any) }
        if supplied["group"] { attrs["consistencygroup_id"] = *group }
        if supplied["name"] { attrs["name"] = *name }
        show(api.CreateRecord(ctx, attrs))
    case "delete":
        receipt, err := api.DeleteRecord(ctx, *identity,
            consistencygroupsnapshots.WithRecordDeleteIgnoreMissing(!*strict))
        if receipt != nil {
            fmt.Printf("actual_http=%d actual_body=%q ignored_missing=%t\n",
                receipt.StatusCode, receipt.Envelope, receipt.IgnoredMissing)
        }
        if err != nil { log.Fatal(err) }
    default:
        log.Fatal("unknown operation: ", *operation)
    }
}
```

```sh
python consistency_snapshots.py list --query '{"limit": 20, "status": "available"}'
go run ./cmd/consistency-snapshots -operation list -query '{"limit": 20, "status": "available"}'
python consistency_snapshots.py create --group source-consistency-group-id --name nightly
go run ./cmd/consistency-snapshots -operation create -group source-consistency-group-id -name nightly
python consistency_snapshots.py find --identity nightly --strict
go run ./cmd/consistency-snapshots -operation find -identity nightly -strict
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2503-L2640)
- [Consistency Group Snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/consistency_group_snapshot.py#L16-L49)
- [Cgsnapshots](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/cgsnapshots.py)
- [default standard_extensions](https://github.com/openstack/cinder/blob/27.0.0/cinder/common/config.py)
- [API](https://github.com/openstack/cinder/blob/27.0.0/cinder/group/api.py)
- [Group Snapshots](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_snapshots.py)
