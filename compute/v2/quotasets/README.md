# Quotasets

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/quotasets.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Reset`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithResetIgnoreMissing`

- DELETE removes quota overrides for the fixed project; a successful bodyless response returns independent headers and no follow-up GET or fabricated quota object.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`.
- Reset defaults to strict 404 handling; only an explicitly ignored 404 becomes nil,nil, the last missing-policy option wins and invalid options fail before HTTP.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/project_quotas_contracts_test.go:TestProjectQuotaInvalidOptionsFailBeforeHTTP`.
- The original HTTP status, body and response header remain in the error chain; 404 is additionally ErrNotFound and context cancellation/deadlines remain observable.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/project_quotas_contracts_test.go:TestProjectQuotaCancellationAndTimeoutPreserveCause`.

Python/Go differences:

- The quota singleton exposes Reset rather than generic resource Delete; success returns ResetResponse(ProjectID, Header) instead of requiring a quota extraction from an empty DELETE body.
- 404 remains an error by default; optional WithResetIgnoreMissing changes only that status.
- UserQuotaScope is an additional high-level mapping for fixed user_id requests; the pinned native API and generated per-project method contract are unchanged.

### `gophercloud:compute/v2/quotasets.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Get`

- Explicit project IDs bind without a lookup; exact names resolve once with a separate Keystone v3 client and do not replace the target with response quota IDs.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaResolvesExactProjectNameWithSeparateKeystoneClientOnce`, `api/project_quotas_contracts_test.go:TestProjectQuotaProjectLookupFailuresPreventNovaRequests`, `connectiontest/connection_quotas_test.go:TestConnectionProjectQuotasSuppliesSeparateIdentityClient`, `connectiontest/connection_quotas_test.go:TestConnectionProjectQuotasNeedsNoIdentityForIDOrAuthScope`.
- Recorded Keystone v2/v3 authentication supplies a frozen project ID; manual, unscoped and malformed results fail without Nova requests.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaCurrentProjectUsesRecordedV3AndV2AuthenticationAndFreezesID`, `api/project_quotas_contracts_test.go:TestProjectQuotaCurrentProjectRejectsUnscopedManualAndInvalidAuthResults`.
- GET /os-quota-sets/{projectID} retains all native limits, zero/unlimited values, the separate request ProjectID and response ID, extension/null fields and independent response headers.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`.
- Missing, null, scalar/array quota objects and invalid known field types fail instead of creating a zero-valued singleton.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaRejectsMalformedQuotaObjects`.
- The original HTTP status, body and response header remain in the error chain; 404 is additionally ErrNotFound and context cancellation/deadlines remain observable.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/project_quotas_contracts_test.go:TestProjectQuotaCancellationAndTimeoutPreserveCause`.

Python/Go differences:

- The Go scope returns an embedded native quota model plus ProjectID, decoded raw quota fields and copied headers instead of a Result extractor; the request project remains fixed even when the response ID differs.
- Malformed singleton envelopes are rejected explicitly; the scope preserves the selected Compute microversion rather than changing it.
- UserQuotaScope is an additional high-level mapping for fixed user_id requests; the pinned native API and generated per-project method contract are unchanged.

### `gophercloud:compute/v2/quotasets.GetDetail`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Detail`

- Explicit project IDs bind without a lookup; exact names resolve once with a separate Keystone v3 client and do not replace the target with response quota IDs.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaResolvesExactProjectNameWithSeparateKeystoneClientOnce`, `api/project_quotas_contracts_test.go:TestProjectQuotaProjectLookupFailuresPreventNovaRequests`, `connectiontest/connection_quotas_test.go:TestConnectionProjectQuotasSuppliesSeparateIdentityClient`, `connectiontest/connection_quotas_test.go:TestConnectionProjectQuotasNeedsNoIdentityForIDOrAuthScope`.
- Recorded Keystone v2/v3 authentication supplies a frozen project ID; manual, unscoped and malformed results fail without Nova requests.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaCurrentProjectUsesRecordedV3AndV2AuthenticationAndFreezesID`, `api/project_quotas_contracts_test.go:TestProjectQuotaCurrentProjectRejectsUnscopedManualAndInvalidAuthResults`.
- GET /os-quota-sets/{projectID}/detail retains native limit/in_use/reserved values, raw nested extensions and independent response headers.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`.
- Missing, null, scalar/array quota objects and invalid known field types fail instead of creating a zero-valued singleton.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaRejectsMalformedQuotaObjects`.
- The original HTTP status, body and response header remain in the error chain; 404 is additionally ErrNotFound and context cancellation/deadlines remain observable.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/project_quotas_contracts_test.go:TestProjectQuotaCancellationAndTimeoutPreserveCause`.

Python/Go differences:

- The Go scope returns an embedded native quota model plus ProjectID, decoded raw quota fields and copied headers instead of a Result extractor; the request project remains fixed even when the response ID differs.
- Malformed singleton envelopes are rejected explicitly; the scope preserves the selected Compute microversion rather than changing it.
- UserQuotaScope is an additional high-level mapping for fixed user_id requests; the pinned native API and generated per-project method contract are unchanged.

### `gophercloud:compute/v2/quotasets.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithUpdateForce`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaOptions`

- PUT /os-quota-sets/{projectID} uses a quota_set envelope; nil limits are omitted, zero and -1 are sent, native force=true and explicit force=false are distinct and the last functional option wins.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`, `api/project_quotas_contracts_test.go:TestProjectQuotaUpdateDistinguishesOmittedFalseZeroAndUnlimited`.
- Extension values are snapshotted; invalid limits, core collisions, unsupported request capabilities and nil options fail before HTTP.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`, `api/project_quotas_contracts_test.go:TestProjectQuotaInvalidOptionsFailBeforeHTTP`.
- The updated server quota response, extension/null fields and independent headers are returned; malformed models fail decoding.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`, `api/project_quotas_contracts_test.go:TestProjectQuotaRejectsMalformedQuotaObjects`.
- The original HTTP status, body and response header remain in the error chain; 404 is additionally ErrNotFound and context cancellation/deadlines remain observable.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/project_quotas_contracts_test.go:TestProjectQuotaCancellationAndTimeoutPreserveCause`.
- All fourteen typed integer limits retain exact values above 2^53 in scope Update; WithQuotaOptions snapshots pointers at construction and remains independent across repeated applications.
  Tests: `api/nova_project_quotas_precision_test.go:TestNovaProjectQuotaTypedLimitsPreserveIntegerPrecision`, `api/nova_project_quotas_precision_test.go:TestNovaProjectQuotaOptionsCapturePointersAndAreReusable`.

Python/Go differences:

- Callers provide native concrete UpdateOpts and SDK-owned functional options rather than UpdateOptsBuilder implementations.
- The scope rejects typed limits below -1 before HTTP, adds an explicit force=false option, and returns QuotaResource with the fixed project and response metadata.
- UserQuotaScope is an additional high-level mapping for fixed user_id requests; the pinned native API and generated per-project method contract are unchanged.

### `python:cloud/_compute/ComputeCloudMixin/delete_compute_quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.FindProjectComputeQuotas`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteComputeQuotas`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Reset`

- delete_compute_quotas strict-finds the project and calls revert_quota_set (bodyless DELETE os-quota-sets/{project}); DeleteComputeQuotas does the same lookup and one DELETE, returning copied response headers.
  Tests: `connectiontest/connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeDeleteQuotasReturnsResetAndVolumeSideWorks`, `connectiontest/connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeQuotaHelpersMissingProjectStopsBeforeService`.
- The strict 404 and HTTP cause preservation of the reset are the mapped revert_quota_set contract.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`.

Python/Go differences:

- Python Resource inputs and **kwargs are replaced by a project name-or-ID string and typed options; the cloud helpers accept only name_or_id as the project input, so no Resource/dict project input exists to map.
- Lookup is the strict find_project sequence (GET projects/{value}, then name-filtered list after a 404) on every call, matching Python; the explicit ProjectQuotas scope remains the lookup-free path for known IDs.
- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding, so there is no separate version check and an Identity endpoint that is not v3 fails at the request.
- Python Proxy.request can serve the lookup GET from a connection response cache; Go has no client-side GET cache.
- Python translates responses into mutable Resources; Go returns the typed quota model with raw fields, headers and status.
- Python returns None; Go returns the reset target and headers.

### `python:cloud/_compute/ComputeCloudMixin/get_compute_quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.FindProjectComputeQuotas`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetComputeQuotas`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Get`

- get_compute_quotas(name_or_id) looks the project up with find_project(ignore_missing=False) and then fetches the quota set without usage; GetComputeQuotas performs the same strict lookup before one quota GET and stops with resource.ErrNotFound and no quota request for a missing project.
  Tests: `connectiontest/connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeQuotaHelpersLookUpIDThenName`, `connectiontest/connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeQuotaHelpersMissingProjectStopsBeforeService`, `connectiontest/connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeQuotaHelpersHonorCanceledContext`.
- The delegated compute get_quota_set (usage=False route, normalized known and unknown quota fields, microversion selection) is separately mapped and the helper reuses it.
  Tests: `connectiontest/connection_quotas_test.go:TestConnectionProjectQuotasSuppliesSeparateIdentityClient`, `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`.

Python/Go differences:

- Python Resource inputs and **kwargs are replaced by a project name-or-ID string and typed options; the cloud helpers accept only name_or_id as the project input, so no Resource/dict project input exists to map.
- Lookup is the strict find_project sequence (GET projects/{value}, then name-filtered list after a 404) on every call, matching Python; the explicit ProjectQuotas scope remains the lookup-free path for known IDs.
- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding, so there is no separate version check and an Identity endpoint that is not v3 fails at the request.
- Python Proxy.request can serve the lookup GET from a connection response cache; Go has no client-side GET cache.
- Python translates responses into mutable Resources; Go returns the typed quota model with raw fields, headers and status.

### `python:cloud/_compute/ComputeCloudMixin/set_compute_quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.FindProjectComputeQuotas`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SetComputeQuotas`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithUpdateForce`

- set_compute_quotas strict-finds the project, forces force=True over any caller value and sends update_quota_set; SetComputeQuotas looks up once and appends force=true after the caller's options, preserving zero values and omitting unset limits.
  Tests: `connectiontest/connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeSetQuotasForcesTrueAndKeepsZero`, `connectiontest/connection_py_cloud_compute_quotas_test.go:TestPyCloudComputeQuotaHelpersMissingProjectStopsBeforeService`.
- Quota update body building (omission, zero, unlimited, explicit extension fields, protected-key rejection) is the mapped update_quota_set contract.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaUpdateDistinguishesOmittedFalseZeroAndUnlimited`, `api/project_quotas_contracts_test.go:TestProjectQuotaInvalidOptionsFailBeforeHTTP`.

Python/Go differences:

- Python Resource inputs and **kwargs are replaced by a project name-or-ID string and typed options; the cloud helpers accept only name_or_id as the project input, so no Resource/dict project input exists to map.
- Lookup is the strict find_project sequence (GET projects/{value}, then name-filtered list after a 404) on every call, matching Python; the explicit ProjectQuotas scope remains the lookup-free path for known IDs.
- Python calls utils.ensure_service_version(identity, '3'); Go has only an Identity v3 binding, so there is no separate version check and an Identity endpoint that is not v3 fails at the request.
- Python Proxy.request can serve the lookup GET from a connection response cache; Go has no client-side GET cache.
- Python translates responses into mutable Resources; Go returns the typed quota model with raw fields, headers and status.
- Python returns None; Go returns the updated quota model. The PUT is sent once and never replayed after an uncertain failure.

### `python:compute/v2/get_quota_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Detail`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.InUser`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.UserQuotaScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.UserQuotaScope.Detail`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaReadQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaUsage`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaReadLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.UserQuotaScope.GetRecord`

- Explicit project IDs bind without a lookup; exact names resolve once with a separate Keystone v3 client and do not replace the target with response quota IDs.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaResolvesExactProjectNameWithSeparateKeystoneClientOnce`, `api/project_quotas_contracts_test.go:TestProjectQuotaProjectLookupFailuresPreventNovaRequests`, `connectiontest/connection_quotas_test.go:TestConnectionProjectQuotasSuppliesSeparateIdentityClient`, `connectiontest/connection_quotas_test.go:TestConnectionProjectQuotasNeedsNoIdentityForIDOrAuthScope`.
- Python usage=False and usage=True endpoint choices have Go Get and Detail counterparts; the native detail triple remains intact.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`.
- The original HTTP status, body and response header remain in the error chain; 404 is additionally ErrNotFound and context cancellation/deadlines remain observable.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/project_quotas_contracts_test.go:TestProjectQuotaCancellationAndTimeoutPreserveCause`.
- The explicit user scope fixes the project/user pair; exact user names resolve once through the inherited or overridden separate Keystone client, explicit IDs need no lookup, and missing, ambiguous or invalid user inputs prevent Nova requests.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserNameUsesInheritedOrOverrideKeystoneClientOnce`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserLookupFailuresDoNotCallNova`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserInvalidInputsAndZeroScopeCannotResetProject`.
- User Get and Detail encode exactly one user_id query, preserve the selected microversion and request headers, retain native limits/detail triples and independent raw big-integer/null fields plus copied headers, and do not expose project-only Defaults or retarget from response identities. HTTP/decode causes and explicit-user context cancellation remain observable.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserOperationsKeepQueryTargetAndRawResponse`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserErrorsKeepBothIdentitiesAndMissingPolicy`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserMalformedResponsesKeepDecodeCauses`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserContextAndCurrentProjectKeepExplicitUser`.
- User reads retain the fixed query through reauthentication, backoff and retry callbacks on the original transport; Get preserves explicit auth headers when the provider has no token and can cancel while shared provider reauthentication is blocked.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRetryAndReauthKeepOneQueryAndExactBody`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserPreservesCallerRedirectRejectionAndExplicitAuthHeaders`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserCancellationDoesNotWaitForSharedReauth`.
- GET project/user quota owns arbitrary Requests query and Source usage endpoint choices without rebinding from response IDs.
  Tests: `compute/v2/quotasets/python_quota_query_test.go:TestNovaQuotaRecordProjectUserUsageAndArbitraryFetchQueries`.
- Default Nova version discovery selects an advertised version up to QuotaSet ceiling2.56 and retains discovery evidence separately from quota GET. Explicit or client-selected versions are uncapped.
  Tests: `compute/v2/quotasets/python_quota_query_test.go:TestNovaQuotaRecordDefaultDiscoverySelectsCeilingAndRetainsReceipts`.
- Explicit empty/version overrides bypass discovery, nil/bound/base_path/requires_id controls fail before HTTP, option inputs are detached and every fetch is uncached.
  Tests: `compute/v2/quotasets/python_quota_query_test.go:TestNovaQuotaRecordExplicitVersionSkipsDiscoveryAndAllowsNoVersion`.
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
- Every option callback is surrounded by sticky captured-source/context/operation guards and independent snapshots; a later callback cannot hide source mutation/cancellation or mutate earlier caller-owned query maps.
  Tests: `internal/quotafetch/read_test.go:TestQuotaOptionsObserveSourceAndCancellationBeforeRestoringCallback`, `internal/quotafetch/read_test.go:TestQuotaEachCallbackBoundaryOwnsOptionsBeforeLaterAliasedMutation`.
- Library-owned descriptor profiles are detached before callbacks, and the final response projection rechecks captured source/context/operation guards. Accepted Close cancellation retains actual receipt and no success Resource.
  Tests: `internal/quotafetch/read_test.go:TestQuotaAcceptedCloseCancellationRetainsActualReceiptWithoutResource`, `internal/quotafetch/read_test.go:TestQuotaProfileDescriptorSliceIsOwnedBeforeCallerOptions`.
- Nova base_path is a bound proxy argument collision by presence, so public WithQuotaReadQuery rejects it without invoking a caller marshaler or doing version discovery/HTTP.
  Tests: `compute/v2/quotasets/python_quota_query_test.go:TestNovaQuotaBasePathPresenceRejectsWithoutInvokingCallerMarshaler`.

Python/Go differences:

- Go returns native typed limit/detail models plus raw fields and copied headers; Python common QuotaSet flattens limit values and separates usage/reservation dictionaries.
- Go references explicitly distinguish IDs from exact project names, and the scope freezes the project once.
- UserQuotaScope requires an explicit user reference and freezes both identities; it does not infer the current user, verify project membership, or promote project-only Defaults. Nova validates authorization and quota policy.
- The user scope owns a guarded HTTP client while retaining caller transport, redirect policy and delegated provider authentication/retry behavior; this narrower tested contract does not establish parity with all inherited Python Adapter behavior.
- New GetRecord provides Source quota normalization and owned inherited fetch controls; existing native Get/Detail/Usage retain earlier reviewed typed models.
- Concrete JSON-domain Go controls validate strings/bools, UTF8 and service-origin routes; arbitrary Python-object Resource attribute/method collisions, Adapter cache/session substitution and dirty commit remain whole-project SDK-R1/C1/S1 work. Fresh GetRecord has no response cache; skip_cache is accepted as an owned non-query control.
- Connection location is explicit immutable CloudLocation input; default is null. Actual server extensions are retained in Wire while fixed URI project and selected microversion own their normalized fields.
- Go InProject preserves the existing single unescaped URL segment identity contract; project IDs with slash/percent are rejected before quota fetch instead of allowing Source-style route interpolation.

### `python:compute/v2/get_quota_set_defaults`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Defaults`

- The distinct /os-quota-sets/{project}/defaults GET sends no user_id query, preserves configured 2.56/auth/custom headers and keeps the scope project separate from the response quota ID; unknown fields, large integers, nulls and independent raw/header snapshots remain available.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaDefaultsPreserveProjectAndRawResponse`.
- Defaults preserves 403/404/unexpected-203 response code/body/header and the Defaults operation cause; only 404 adds ErrNotFound. Missing/null/array/scalar quota_set envelopes and wrong native limit types fail with the original decode cause.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaDefaultsPreserveHTTPAndDecodeErrors`.
- A pre-canceled Defaults call makes no request, and a deadline after the request starts remains context.DeadlineExceeded.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaDefaultsRespectContext`.
- With default arguments Defaults sends one GET /os-quota-sets/{project}/defaults without a query and decodes the quota_set limits, matching Python QuotaSet.fetch with the defaults base_path.
  Tests: `compute/v2/quotasets/python_rejudge_test.go:TestPythonRejudgeComputeQuotaDefaultsRequest`.

Python/Go differences:

- Go returns native typed limits plus independent raw fields and copied headers on a fixed project scope instead of mutating a Python common QuotaSet Resource.
- The Go scope retains the selected Compute microversion; it does not automatically impose the Python QuotaSet maximum microversion 2.56. That maximum is not a minimum API requirement.
- Python accepts a project ID or Project Resource; Go binds the project once with quotasets.API.InProject (an ID needs no lookup).
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Go has no client-side GET response cache or skip_cache control; every call sends its request.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:compute/v2/revert_quota_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Reset`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.InUser`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.UserQuotaScope.Reset`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithResetQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithResetIgnoreMissing`

- WithResetQuery adds revert_quota_set(**query) parameters as literal values to the project DELETE os-quota-sets/{project} and next to the fixed user_id on the user scope, a later value for a key wins, a 404 stays an error that is also resource.ErrNotFound unless WithResetIgnoreMissing(true) is given, and a 500 stays a Gophercloud status error inside resource.OperationError.
  Tests: `compute/v2/quotasets/python_reset_query_test.go:TestPythonQuotaResetForwardsQueryAndMissingPolicy`.
- WithResetQuery rejects user_id, a blank or control-character key, an invalid UTF-8 value and a nil option on both scopes before HTTP.
  Tests: `compute/v2/quotasets/python_reset_query_test.go:TestPythonQuotaResetQueryRejectsScopeOwnedAndInvalidKeys`.
- The no-query project reset uses bodyless DELETE and strict missing handling without an additional GET.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`, `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`.
- The original HTTP status, body and response header remain in the error chain; 404 is additionally ErrNotFound and context cancellation/deadlines remain observable.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/project_quotas_contracts_test.go:TestProjectQuotaCancellationAndTimeoutPreserveCause`.
- User Reset sends bodyless DELETE with exactly one fixed user_id, accepts 202/204 without a follow-up request, rejects unexpected success codes, preserves both identities and HTTP causes, and ignores only an explicitly permitted 404. Invalid or zero scopes cannot fall back to project-wide Reset.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserOperationsKeepQueryTargetAndRawResponse`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserResetAccepts202And204WithoutFollowUp`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserErrorsKeepBothIdentitiesAndMissingPolicy`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserInvalidInputsAndZeroScopeCannotResetProject`.
- Reset redirect fixtures reject dropped, changed or duplicate user queries, changed project/origin/method, opaque URLs and virtual-host mutation before a second unsafe request. Same-pair redirects retain the caller policy; its rejection remains in the error chain.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRedirectCannotChangeProjectUserOriginOrMethod`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRedirectKeepsSamePairAndCallerPolicy`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserPreservesCallerRedirectRejectionAndExplicitAuthHeaders`.
- User Reset retains its pair and single query through reauthentication/backoff/retry, observes canceled and expired contexts, and preserves caller and HTTP timeout cancellation even when redirect policy replaces the request context.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRetryAndReauthKeepOneQueryAndExactBody`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserContextAndCurrentProjectKeepExplicitUser`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRedirectRetainsCallerAndHTTPTimeoutContexts`.

Python/Go differences:

- Python urlencodes str(value) for each query value; WithResetQuery takes the literal text, so the caller passes "True" or a Python list spelling to reproduce those values.
- Python accepts every status below 400; Go accepts Nova's 202 and 204.
- Python sends min(QuotaSet _max_microversion 2.56, server maximum); Go sends the client microversion.
- Python passes user_id as a query keyword; Go maps it to UserQuotaScope and rejects user_id in WithResetQuery. Go encodes query keys in sorted order.
- Python returns None; Go returns ResetResponse with the fixed project/user identities and copied headers and can explicitly ignore a 404.
- UserQuotaScope is separate from ProjectQuotaScope, so a user Reset cannot promote a project-wide operation; guarded redirects reject changes to the exact request target rather than forwarding arbitrary query mutations.

### `python:compute/v2/update_quota_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithUpdateForce`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.InUser`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.UserQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaOptions`

- Concrete project limit updates distinguish omission, zero, unlimited and explicit force=false, with a SDK-owned body builder and protected extension fields.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaUpdateDistinguishesOmittedFalseZeroAndUnlimited`, `api/project_quotas_contracts_test.go:TestProjectQuotaInvalidOptionsFailBeforeHTTP`.
- The original HTTP status, body and response header remain in the error chain; 404 is additionally ErrNotFound and context cancellation/deadlines remain observable.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/project_quotas_contracts_test.go:TestProjectQuotaCancellationAndTimeoutPreserveCause`.
- The modern optional-user branch has an explicit fixed UserQuotaScope counterpart: exact user resolution uses the separate Keystone client once, PUT encodes one user_id, response identities cannot retarget the scope, and lookup or invalid-input failures make no Nova request.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserNameUsesInheritedOrOverrideKeystoneClientOnce`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserLookupFailuresDoNotCallNova`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserOperationsKeepQueryTargetAndRawResponse`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserInvalidInputsAndZeroScopeCannotResetProject`.
- User Update reuses the SDK-owned concrete quota builder, preserves zero/unlimited and explicit force=false, snapshots extension input and independent response raw/header data, and retains exact integer PUT bodies and one fixed query across reauthentication, backoff and retries.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserOperationsKeepQueryTargetAndRawResponse`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRetryAndReauthKeepOneQueryAndExactBody`.
- User Update preserves HTTP status/body/header and pair-aware missing errors, decode causes and context cancellation/deadlines; same-pair redirects retain the caller policy without adding a second user_id.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserErrorsKeepBothIdentitiesAndMissingPolicy`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserMalformedResponsesKeepDecodeCauses`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserContextAndCurrentProjectKeepExplicitUser`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRedirectKeepsSamePairAndCallerPolicy`.
- A project-scope Update with supplied limits sends one PUT /os-quota-sets/{project} whose quota_set body holds only those limits, like Python's dirty-attribute commit, and decodes the returned limits; an empty update sends an empty quota_set.
  Tests: `compute/v2/quotasets/python_rejudge_test.go:TestPythonRejudgeComputeQuotaUpdateRequest`.

Python/Go differences:

- Go uses explicit pointer-valued native inputs and functional options instead of Resource dirty-field commits and arbitrary attrs.
- The Go scope returns QuotaResource retaining native values/raw decoded fields/header; Python returns a mutable normalized QuotaSet.
- Python modern update accepts an optional user and attrs; Go obtains an explicit UserQuotaScope before Update and does not infer membership or permit options to replace its fixed user_id.
- User request guards and concrete snapshot options are additional high-level policies; they do not implement Python legacy quota objects, Resource dirty state or arbitrary commit/query behavior.
- When no attribute is supplied Python skips HTTP and returns its local QuotaSet; Go sends PUT {"quota_set":{}}, which Nova answers with the unchanged limits.
- The pinned Python user= branch passes user_id to Resource.commit, whose _prepare_request does not accept it (TypeError); Go UserQuotaScope.Update sends the PUT with ?user_id=.
- The deprecated Python QuotaSet/project_id overload sends the same PUT; Go expresses it through InProject(project).Update.
- Python QuotaSet caps the negotiated microversion at 2.56; Go sends the configured Compute microversion.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := quotasets.New(computeClient)
scope, err := api.InProject(ctx, resource.ID("project-id"))
if err != nil { return err }
limits, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(scope.ProjectID(), limits.Cores, limits.Header.Get("X-Openstack-Request-Id"))

usage, err := scope.Detail(ctx)
if err != nil { return err }
fmt.Println(usage.Cores.Limit, usage.Cores.InUse, usage.Cores.Reserved)

zero, unlimited := 0, -1
updated, err := scope.Update(ctx, quotasets.UpdateOpts{
    Cores: &unlimited,
    Instances: &zero,
}, quotasets.WithUpdateForce(false))
if err != nil { return err }
fmt.Println(updated.Cores, updated.Instances)

reset, err := scope.Reset(ctx)
if err != nil { return err }
fmt.Println(reset.Header.Get("X-Openstack-Request-Id"))
```

```go
userScope, err := scope.InUser(ctx, resource.ID("user-id"))
if err != nil { return err }
userLimits, err := userScope.Get(ctx)
if err != nil { return err }
fmt.Println(userLimits.ProjectID, userLimits.UserID, userLimits.Cores)

zero := 0
_, err = userScope.Update(ctx, quotasets.UpdateOpts{Cores: &zero},
    quotasets.WithUpdateForce(false))
if err != nil { return err }
_, err = userScope.Reset(ctx)
if err != nil { return err }
```

```go
cores := 7
snapshot := quotasets.WithQuotaOptions(quotasets.UpdateOpts{Cores: &cores})
cores = 99
_, err := scope.Update(ctx, quotasets.UpdateOpts{}, snapshot) // cores=7
if err != nil { return err }
```

## References

- [Compute](https://docs.openstack.org/api-ref/compute/#quota-sets-os-quota-sets)
- [Compute](https://docs.openstack.org/api-ref/compute/#update-quotas)
- [Nova version history](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html)
- [compute proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [cloud quota helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py)
- [common QuotaSet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [native quota API](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/quotasets/requests.go)
- [Project Quotas Contracts Test](../../../api/project_quotas_contracts_test.go)
- [Nova Project Quotas Precision Test](../../../api/nova_project_quotas_precision_test.go)
- [Nova Quota Defaults Users Test](../../../api/nova_quota_defaults_users_test.go)
- [Python Quota Class](python-quota-class.md)
