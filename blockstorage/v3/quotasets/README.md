# Quotasets

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/v3/quotasets.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Get`

- Explicit project IDs bind without HTTP; exact project names resolve once through a separate Keystone v3 client across all pages, with missing, ambiguous, invalid-ID and authorization failures stopping Cinder calls.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaResolvesExactProjectNameWithSeparateKeystoneClientOnce`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaProjectLookupFailuresPreventCinderRequests`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- Recorded Keystone v2 tenant or v3 project IDs are frozen when the scope is created; manual, unsupported, typed-nil, system/domain and malformed authentication results are rejected without quota requests or token disclosure.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCurrentProjectUsesRecordedV3AndV2AuthenticationAndFreezesID`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCurrentProjectRejectsUnscopedManualAndInvalidAuthResults`, `connection_project_quotas_test.go:TestConnectionServiceQuotasIDAuthAndCancellation`.
- GET uses /os-quota-sets/{fixedProject}, preserves native zero/unlimited limits, separate response ID, per-type Extra, exact raw extension/null/large-integer fields and copied response headers.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`.
- Missing, null, array/scalar quota_set envelopes and invalid known field types return decoding errors rather than zero-valued quota objects.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaRejectsMalformedQuotaObjects`.
- All five operations retain original HTTP status/body/header causes, add ErrNotFound only for 404 and preserve pre-canceled contexts; in-flight GET cancellation and Usage deadlines remain observable.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCancellationAndTimeoutPreserveCause`.

Python/Go differences:

- ProjectQuotaScope fixes one project and returns a typed native model plus ProjectID, raw Body and independent Header metadata; callers do not construct native builders or extract Results.

### `gophercloud:blockstorage/v3/quotasets.GetDefaults`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Defaults`

- Explicit project IDs bind without HTTP; exact project names resolve once through a separate Keystone v3 client across all pages, with missing, ambiguous, invalid-ID and authorization failures stopping Cinder calls.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaResolvesExactProjectNameWithSeparateKeystoneClientOnce`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaProjectLookupFailuresPreventCinderRequests`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- Recorded Keystone v2 tenant or v3 project IDs are frozen when the scope is created; manual, unsupported, typed-nil, system/domain and malformed authentication results are rejected without quota requests or token disclosure.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCurrentProjectUsesRecordedV3AndV2AuthenticationAndFreezesID`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCurrentProjectRejectsUnscopedManualAndInvalidAuthResults`, `connection_project_quotas_test.go:TestConnectionServiceQuotasIDAuthAndCancellation`.
- GET uses the distinct /os-quota-sets/{fixedProject}/defaults endpoint, retains server defaults and raw per-type/large-integer fields, and leaves configured overrides unchanged.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`.
- Missing, null, array/scalar quota_set envelopes and invalid known field types return decoding errors rather than zero-valued quota objects.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaRejectsMalformedQuotaObjects`.
- All five operations retain original HTTP status/body/header causes, add ErrNotFound only for 404 and preserve pre-canceled contexts; in-flight GET cancellation and Usage deadlines remain observable.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCancellationAndTimeoutPreserveCause`.

Python/Go differences:

- ProjectQuotaScope fixes one project and returns a typed native model plus ProjectID, raw Body and independent Header metadata; callers do not construct native builders or extract Results.

### `gophercloud:blockstorage/v3/quotasets.GetUsage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Usage`

- Explicit project IDs bind without HTTP; exact project names resolve once through a separate Keystone v3 client across all pages, with missing, ambiguous, invalid-ID and authorization failures stopping Cinder calls.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaResolvesExactProjectNameWithSeparateKeystoneClientOnce`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaProjectLookupFailuresPreventCinderRequests`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- Recorded Keystone v2 tenant or v3 project IDs are frozen when the scope is created; manual, unsupported, typed-nil, system/domain and malformed authentication results are rejected without quota requests or token disclosure.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCurrentProjectUsesRecordedV3AndV2AuthenticationAndFreezesID`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCurrentProjectRejectsUnscopedManualAndInvalidAuthResults`, `connection_project_quotas_test.go:TestConnectionServiceQuotasIDAuthAndCancellation`.
- GET uses ?usage=true on the project endpoint, retains native Limit/InUse/Reserved/Allocated values and exact raw nested extension fields rather than Nova detail routing.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`.
- Missing, null, array/scalar quota_set envelopes and invalid known field types return decoding errors rather than zero-valued quota objects.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaRejectsMalformedQuotaObjects`.
- All five operations retain original HTTP status/body/header causes, add ErrNotFound only for 404 and preserve pre-canceled contexts; in-flight GET cancellation and Usage deadlines remain observable.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCancellationAndTimeoutPreserveCause`.

Python/Go differences:

- ProjectQuotaScope fixes one project and returns a typed native model plus ProjectID, raw Body and independent Header metadata; callers do not construct native builders or extract Results.

### `gophercloud:blockstorage/v3/quotasets.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithUpdateForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithVolumeTypeQuota`

- PUT updates only supplied limits: nil omits, zero and -1 remain explicit, native force=true is retained and WithUpdateForce(false) sends false with last-option precedence.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaUpdateDistinguishesOmittedFalseZeroAndUnlimited`, `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`.
- WithQuotaOptions snapshots typed pointers and nested Extra maps/slices at option creation and remains reusable; extension options are detached, omitted core fields are still protected and duplicate Extra/field keys fail before HTTP.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaDeepOptionSnapshotAndCoreCollisions`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaInvalidOptionsFailBeforeHTTP`.
- Raw-JSON request construction preserves all seven typed integer limits and per-type gigabyte values above 2^53 on 64-bit platforms; volume-type quota keys use the exact supplied type name without lookup.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaTypedLimitsRetainIntegerPrecision`, `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`.
- Only native HTTP 200 succeeds for PUT; wrong success codes retain their original expected/actual status and response headers.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaDeleteAndUpdateAcceptOnlyNative200`.
- Missing, null, array/scalar quota_set envelopes and invalid known field types return decoding errors rather than zero-valued quota objects.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaRejectsMalformedQuotaObjects`.
- All five operations retain original HTTP status/body/header causes, add ErrNotFound only for 404 and preserve pre-canceled contexts; in-flight GET cancellation and Usage deadlines remain observable.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCancellationAndTimeoutPreserveCause`.

Python/Go differences:

- The concrete scope API owns extension construction and protects core fields that native Extra can overwrite; malformed options and typed limits below -1 fail before HTTP.
- WithQuotaOptions copies at option construction; direct opts and native WithUpdateOptions are detached during Update. WithUpdateForce preserves explicit false; raw JSON avoids native builder float64 rounding.

### `gophercloud:blockstorage/v3/quotasets.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Reset`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithResetIgnoreMissing`

- DELETE resets overrides at the fixed project endpoint, accepts native HTTP 200, returns bodyless reset metadata and performs no follow-up GET.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`, `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaDeleteAndUpdateAcceptOnlyNative200`.
- Reset is strict on 404 by default; explicit ignore-missing suppresses only 404, a later false restores strict behavior and nil options are rejected before HTTP.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaInvalidOptionsFailBeforeHTTP`.
- All five operations retain original HTTP status/body/header causes, add ErrNotFound only for 404 and preserve pre-canceled contexts; in-flight GET cancellation and Usage deadlines remain observable.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCancellationAndTimeoutPreserveCause`.

Python/Go differences:

- The project singleton exposes Reset rather than a generic resource Delete, with ResetResponse(ProjectID, Header) instead of a fabricated quota model.
- Explicit ignore-missing affects only 404; the default matches the native strict error contract.

### `python:block_storage/v3/get_quota_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Usage`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaReadQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaUsage`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaReadLocation`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaReadBasePath`

- The ordinary and usage=true Cinder endpoint forms have distinct Go methods preserving limit and allocation triples plus raw response data.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`.
- All five operations retain original HTTP status/body/header causes, add ErrNotFound only for 404 and preserve pre-canceled contexts; in-flight GET cancellation and Usage deadlines remain observable.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCancellationAndTimeoutPreserveCause`.
- Default Cinder sends usage=False and usage=True stays on the same project route; known and unknown quota limits/usage/reservations are normalized with raw allocations retained.
  Tests: `blockstorage/v3/quotasets/python_query_test.go:TestCinderQuotaRecordUsageDefaultAndNestedNormalization`.
- Inherited fetch controls are consumed rather than sent as query; explicit same-origin effective-base routes and selected versions are owned without auto discovery.
  Tests: `blockstorage/v3/quotasets/python_query_test.go:TestCinderQuotaRecordFetchParametersAndBasePathControls`.
- Scope/query/options ownership is fixed across fresh uncached requests; caller inputs cannot mutate options and invalid bound controls fail before HTTP.
  Tests: `blockstorage/v3/quotasets/python_query_test.go:TestCinderQuotaRecordBoundParamsAndOptionOwnership`.
- Source common quota translator owns known int/bool descriptor conversion and ordered limit/usage/reservation normalization, raw unknown fields, independent Resource/Wire/envelope/header metadata and fixed URI/location projections.
  Tests: `internal/quotafetch/read_test.go:TestQuotaRecordNormalizesUsageReservationsAndKnownDescriptors`, `internal/quotafetch/read_test.go:TestQuotaIntegerDescriptorUsesSourceConversionAndExactNumbers`, `internal/quotafetch/read_test.go:TestQuotaLocationUsesCapturedProjectAndIndependentCloudMetadata`.
- Accepted200-399 bodyless/non-JSON and bare-object responses preserve actual evidence; malformed quota shapes and normalization errors retain exact accepted receipt without synthesizing Resource success.
  Tests: `internal/quotafetch/read_test.go:TestQuotaFreshEmptyNonJSONAndBareObjectResponses`, `internal/quotafetch/read_test.go:TestQuotaWrongShapeAndDictionaryOrderFailuresRetainActualResponse`.
- Requests JSON-domain query encoding owns snapshots, nil/default/control classification and last functional options. Bound controls, invalid routes and requires_id are preflight failures.
  Tests: `internal/quotafetch/read_test.go:TestQuotaQueriesOwnJSONInputsLastOptionsAndFetchControls`, `internal/quotafetch/read_test.go:TestQuotaPreflightRejectsScopeCollisionsNilOptionsAndRoutes`.
- Physical read/Close failures retain native401 pointer ErrOriginal/ErrReauth and all causes with one physical response and zero retry/reauth hooks; clean401/503 retain delegated Provider retries and captured query/version.
  Tests: `internal/quotafetch/read_test.go:TestQuotaPhysicalFaultsKeepNative401OriginalAndAllCausesWithoutHooks`, `internal/quotafetch/read_test.go:TestQuotaCleanReauthenticationAndRetryKeepTokenTargetAndVersion`.
- Captured service source/version and request policy stay fixed through callbacks; cancellation retains cause and no invented response.
  Tests: `internal/quotafetch/read_test.go:TestQuotaRetryCannotReplaceSelectedVersionOrRemoveIt`, `internal/quotafetch/read_test.go:TestQuotaCapturedSourceAndCancellationPreventHTTP`.
- Cinder usage defaults to False/True for concrete bool options, but raw Source usage preserves literal Requests null/number/string/list encoding instead of Nova endpoint truthiness.
  Tests: `internal/quotafetch/read_test.go:TestCinderQuotaRawUsageRetainsLiteralRequestsValues`.
- Every option callback is surrounded by sticky captured-source/context/operation guards and independent snapshots; a later callback cannot hide source mutation/cancellation or mutate earlier caller-owned query maps.
  Tests: `internal/quotafetch/read_test.go:TestQuotaOptionsObserveSourceAndCancellationBeforeRestoringCallback`, `internal/quotafetch/read_test.go:TestQuotaEachCallbackBoundaryOwnsOptionsBeforeLaterAliasedMutation`.
- Library-owned descriptor profiles are detached before callbacks, and the final response projection rechecks captured source/context/operation guards. Accepted Close cancellation retains actual receipt and no success Resource.
  Tests: `internal/quotafetch/read_test.go:TestQuotaAcceptedCloseCancellationRetainsActualReceiptWithoutResource`, `internal/quotafetch/read_test.go:TestQuotaProfileDescriptorSliceIsOwnedBeforeCallerOptions`.

Python/Go differences:

- Go preserves native Limit/InUse/Reserved/Allocated structures; Python common QuotaSet flattens each limit and builds separate usage/reservation dictionaries.
- Go requires an explicit ID or name reference and fixes the project once instead of accepting a Python Project object.
- New GetRecord provides Source quota normalization and owned inherited fetch controls; existing native Get/Detail/Usage retain earlier reviewed typed models.
- Concrete JSON-domain Go controls validate strings/bools, UTF8 and service-origin routes; arbitrary Python-object Resource attribute/method collisions, Adapter cache/session substitution and dirty commit remain whole-project SDK-R1/C1/S1 work. Fresh GetRecord has no response cache; skip_cache is accepted as an owned non-query control.
- Connection location is explicit immutable CloudLocation input; default is null. Actual server extensions are retained in Wire while fixed URI project and selected microversion own their normalized fields.
- Go InProject preserves the existing single unescaped URL segment identity contract; project IDs with slash/percent are rejected before quota fetch instead of allowing Source-style route interpolation.

### `python:block_storage/v3/get_quota_set_defaults`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Defaults`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.InProject`

- The dedicated defaults endpoint returns server defaults without resetting or fetching current overrides.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`.
- Missing, null, array/scalar quota_set envelopes and invalid known field types return decoding errors rather than zero-valued quota objects.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaRejectsMalformedQuotaObjects`.
- All five operations retain original HTTP status/body/header causes, add ErrNotFound only for 404 and preserve pre-canceled contexts; in-flight GET cancellation and Usage deadlines remain observable.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaCancellationAndTimeoutPreserveCause`.
- get_quota_set_defaults(project) maps to InProject(resource.ID(project)) and ProjectQuotaScope.Defaults, which sends GET /os-quota-sets/<project>/defaults without query and decodes the default limits and per-volume-type keys.
  Tests: `blockstorage/v3/quotasets/python_rejudge_test.go:TestPythonRejudgeQuotaSetDefaults`.

Python/Go differences:

- Go returns an independent typed/raw response object associated with a fixed project scope rather than mutating a Python QuotaSet Resource.
- Python accepts a Project Resource or a project ID; Go fixes the project once with API.InProject (resource.ID needs no request).
- Python normalizes nested limit/in_use/reserved values into a QuotaSet Resource with usage and reservation attributes; Go decodes the typed QuotaSet fields and keeps every quota field, including per-volume-type keys, as raw JSON in Body.
- Go has no client-side GET response cache; every call sends the request.
- QuotaSet declares no maximum microversion, so Python sends no negotiated version; Go sends the configured client microversion when one is set.

### `python:block_storage/v3/revert_quota_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Reset`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithResetIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.RevertRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaRevertQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaRevertOpts`

- Quota reset uses strict 404 handling by default and sends no follow-up GET; optional ignore-missing changes only 404.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`, `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`.
- Delete params preserve Source urlencode without doseq: None/list/dict/boolean values each become one Python string and exact integers remain precise.
  Tests: `blockstorage/v3/quotasets/python_revert_test.go:TestPythonCinderRevertUsesSourceURLencodeWithoutDoseq`.
- Raw query spellings for route, headers, version and project stay literal params and cannot replace the bound singleton transport.
  Tests: `blockstorage/v3/quotasets/python_revert_test.go:TestPythonCinderRevertRawControlNamesStayDeletionParams`.
- Options own JSON/maps, preserve insertion order and distinguish separate typed transport header/version conveniences.
  Tests: `blockstorage/v3/quotasets/python_revert_test.go:TestPythonCinderRevertOwnsOptionsOrderAndTransportHeaders`.
- Source below-400 bodyless ACK acceptance retains opaque evidence; 404 remains strict and preserves native failure and operation context.
  Tests: `blockstorage/v3/quotasets/python_revert_test.go:TestPythonCinderRevertAcceptsBelow400AndKeeps404Strict`.
- Invalid options, bound source changes and cancellation stop before HTTP or later callbacks.
  Tests: `blockstorage/v3/quotasets/python_revert_test.go:TestPythonCinderRevertPreflightGuardsAndCancellation`.
- Physical accepted/rejected reads and Close errors retain evidence and stop retries or missing-resource suppression.
  Tests: `blockstorage/v3/quotasets/python_revert_test.go:TestPythonCinderRevertPhysicalErrorsNeverReplayOrIgnore404`.
- Native Reset keeps its 200-only contract while the new Source-shaped revert owns its separate acceptance profile.
  Tests: `blockstorage/v3/quotasets/python_revert_test.go:TestPythonCinderRevertKeepsNativeResetSeparate`.
- Configured clean retries preserve the exact bodyless deletion request; native hooks cannot replace its payload.
  Tests: `blockstorage/v3/quotasets/python_revert_test.go:TestPythonCinderRevertCleanRetryKeepsQueryAndRejectsBodyReplacement`.

Python/Go differences:

- Native Reset remains 200-only. Source-shaped RevertRecord accepts below-400 ACKs and returns an owned raw receipt whereas Python revert_quota_set returns None.
- A fixed scalar project scope and concrete JSON/options replace arbitrary mutable Project/QuotaSet objects. Python Resource dirty/cache lifecycle is not emulated; no prefetch or follow-up fetch is added.
- Bulk Go query maps use deterministic sorted insertion order. Raw query values use the reviewed finite JSON PythonString profile; typed Headers/Microversion are explicit Go transport conveniences separate from literal Python **query params.

### `python:block_storage/v3/update_quota_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithUpdateForce`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithVolumeTypeQuota`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.InProject`

- Typed limit omission/zero/-1, force false and exact per-type keys have concrete Go options with detached request data.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaUpdateDistinguishesOmittedFalseZeroAndUnlimited`, `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaDeepOptionSnapshotAndCoreCollisions`, `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaTypedLimitsRetainIntegerPrecision`.
- update_quota_set(project, volumes=5, gigabytes=0, volumes_lvm=3, force=True) maps to ProjectQuotaScope.Update with typed limits, WithVolumeTypeQuota and WithUpdateForce(true) (or WithUpdateField for other keys), sending PUT /os-quota-sets/<project> {"quota_set":{...}} with only those keys.
  Tests: `blockstorage/v3/quotasets/python_rejudge_test.go:TestPythonRejudgeQuotaSetUpdate`.

Python/Go differences:

- Go protects core fields and does not require callers to implement a builder or track dirty Resource state; response metadata is returned independently.
- Python skips the PUT when no attribute is dirty; Go Update with empty options still sends PUT {"quota_set":{}}.
- The deprecated QuotaSet or project_id signature only warns and drops the query argument, sending the same PUT; Go takes the project once with API.InProject.
- Python never sends reservation or usage; Go has no input for them.
- Go cannot send JSON null for a typed limit; Cinder rejects null limits.
- Python normalizes nested limit/in_use/reserved values into a QuotaSet Resource with usage and reservation attributes; Go decodes the typed QuotaSet fields and keeps every quota field, including per-volume-type keys, as raw JSON in Body.
- QuotaSet declares no maximum microversion, so Python sends no negotiated version; Go sends the configured client microversion when one is set.

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.FindProjectBlockStorageQuotas`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetBlockStorageQuotas`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Get`

- get_volume_quotas strict-finds the project then reads the quota set without usage; GetBlockStorageQuotas looks up once and sends one GET, failing before Cinder for a missing project.
  Tests: `connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeDeleteQuotasReturnsResetAndVolumeSideWorks`, `connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeQuotaHelpersMissingProjectStopsBeforeService`.
- Normalized known and unknown quota fields and usage/limit triples are the mapped block storage get_quota_set contract.
  Tests: `api/cinder_project_quotas_responses_test.go:TestCinderProjectQuotaLimitsDefaultsUsageUpdateAndResetKeepFixedTarget`.

Python/Go differences:

- Python Resource inputs and **kwargs are replaced by a project name-or-ID string and typed options; the cloud helpers accept only name_or_id as the project input, so no Resource/dict project input exists to map.
- Lookup is the strict find_project sequence (GET projects/{value}, then name-filtered list after a 404) on every call, matching Python; the explicit ProjectQuotas scope remains the lookup-free path for known IDs.
- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding, so there is no separate version check and an Identity endpoint that is not v3 fails at the request.
- Python Proxy.request can serve the lookup GET from a connection response cache; Go has no client-side GET cache.
- Python translates responses into mutable Resources; Go returns the typed quota model with raw fields, headers and status.
- Block storage in Go is Cinder v3 only; the Python helper dispatches over the configured v2 or v3 proxy.

### `python:cloud/_block_storage/BlockStorageCloudMixin/set_volume_quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.FindProjectBlockStorageQuotas`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SetBlockStorageQuotas`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Update`

- set_volume_quotas strict-finds the project and sends update_quota_set without a force flag; SetBlockStorageQuotas looks up once, sends one PUT and adds no force.
  Tests: `connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeDeleteQuotasReturnsResetAndVolumeSideWorks`, `connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeQuotaHelpersMissingProjectStopsBeforeService`.
- Cinder update body building and error evidence are the mapped update_quota_set contract.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`.

Python/Go differences:

- Python Resource inputs and **kwargs are replaced by a project name-or-ID string and typed options; the cloud helpers accept only name_or_id as the project input, so no Resource/dict project input exists to map.
- Lookup is the strict find_project sequence (GET projects/{value}, then name-filtered list after a 404) on every call, matching Python; the explicit ProjectQuotas scope remains the lookup-free path for known IDs.
- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding, so there is no separate version check and an Identity endpoint that is not v3 fails at the request.
- Python Proxy.request can serve the lookup GET from a connection response cache; Go has no client-side GET cache.
- Python translates responses into mutable Resources; Go returns the typed quota model with raw fields, headers and status.
- Block storage in Go is Cinder v3 only; the Python helper dispatches over the configured v2 or v3 proxy.
- Python returns None; Go returns the updated quota model.

### `python:cloud/_block_storage/BlockStorageCloudMixin/delete_volume_quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.FindProjectBlockStorageQuotas`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteBlockStorageQuotas`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.ProjectQuotaScope.Reset`

- delete_volume_quotas strict-finds the project and calls revert_quota_set; DeleteBlockStorageQuotas looks up once and sends one DELETE.
  Tests: `connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeDeleteQuotasReturnsResetAndVolumeSideWorks`, `connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeQuotaHelpersMissingProjectStopsBeforeService`.
- The reset 404/HTTP cause contract is the mapped revert_quota_set behavior.
  Tests: `api/cinder_project_quotas_contracts_test.go:TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`.

Python/Go differences:

- Python Resource inputs and **kwargs are replaced by a project name-or-ID string and typed options; the cloud helpers accept only name_or_id as the project input, so no Resource/dict project input exists to map.
- Lookup is the strict find_project sequence (GET projects/{value}, then name-filtered list after a 404) on every call, matching Python; the explicit ProjectQuotas scope remains the lookup-free path for known IDs.
- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding, so there is no separate version check and an Identity endpoint that is not v3 fails at the request.
- Python Proxy.request can serve the lookup GET from a connection response cache; Go has no client-side GET cache.
- Python translates responses into mutable Resources; Go returns the typed quota model with raw fields, headers and status.
- Block storage in Go is Cinder v3 only; the Python helper dispatches over the configured v2 or v3 proxy.
- Python returns None (its docstring promises the deleted quota set); Go returns the reset response.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := quotasets.New(volumeClient)
scope, err := api.InProject(ctx, resource.ID("project-id"))
if err != nil { return err }

limits, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(scope.ProjectID(), limits.Volumes, limits.Header.Get("X-Openstack-Request-Id"))

defaults, err := scope.Defaults(ctx)
if err != nil { return err }
fmt.Println(defaults.Volumes)

usage, err := scope.Usage(ctx)
if err != nil { return err }
fmt.Println(usage.Volumes.Limit, usage.Volumes.InUse, usage.Volumes.Reserved)

zero, unlimited := 0, -1
updated, err := scope.Update(ctx, quotasets.UpdateOpts{
    Volumes: &unlimited,
    Snapshots: &zero,
}, quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeVolumes, "SSD", 12))
if err != nil { return err }
fmt.Println(updated.Volumes, string(updated.Body["volumes_SSD"]))

reset, err := scope.Reset(ctx)
if err != nil { return err }
fmt.Println(reset.Header.Get("X-Openstack-Request-Id"))
```

```go
limit := 5
nested := map[string]any{"tier": "original"}
snapshot := quotasets.WithQuotaOptions(quotasets.UpdateOpts{
    Volumes: &limit,
    Extra: map[string]any{"vendor_quota": nested},
})
limit = 99
nested["tier"] = "changed"
_, err := scope.Update(ctx, quotasets.UpdateOpts{}, snapshot) // volumes=5, original
if err != nil { return err }
```

## References

- [Cinder quota API](https://docs.openstack.org/api-ref/block-storage/v3/#quota-sets-extension-os-quota-sets)
- [block storage proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [quota model](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/quota_set.py)
- [common QuotaSet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [cloud quota helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py)
- [Gophercloud v2.15.0 requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/quotasets/requests.go)
- [Cinder Project Quotas Contracts Test](../../../api/cinder_project_quotas_contracts_test.go)
- [Cinder Project Quotas Responses Test](../../../api/cinder_project_quotas_responses_test.go)
- [Python Quota Class](python-quota-class.md)
- [Python Revert](python-revert.md)
