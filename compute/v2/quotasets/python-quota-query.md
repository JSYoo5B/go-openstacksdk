# Python Quota Query

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaResolvesExactProjectNameWithSeparateKeystoneClientOnce`, `api/project_quotas_contracts_test.go:TestProjectQuotaProjectLookupFailuresPreventNovaRequests`, `connection_quotas_test.go:TestConnectionProjectQuotasSuppliesSeparateIdentityClient`, `connection_quotas_test.go:TestConnectionProjectQuotasNeedsNoIdentityForIDOrAuthScope`.
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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
quota = conn.compute.get_quota_set(
    "project-id", usage=True, user_id="user-id", fields=["cores", "ram"]
)
print(quota.cores, quota.usage, quota.reservation)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    scope, err := conn.ProjectQuotas(ctx, resource.ID("project-id"))
    if err != nil { log.Fatal(err) }
    user, err := scope.InUser(ctx, resource.ID("user-id"))
    if err != nil { log.Fatal(err) }
    quota, err := user.GetRecord(ctx,
        quotasets.WithQuotaUsage(true),
        quotasets.WithQuotaReadQuery("fields", []string{"cores", "ram"}),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(quota.Resource.Body["cores"]))
    fmt.Println(string(quota.Resource.Body["usage"]), string(quota.Resource.Body["reservation"]))
    fmt.Println(quota.ProjectID, quota.UserID, quota.StatusCode)
}
```

## References

- [compute proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [Quota Sets](https://github.com/openstack/nova/blob/31.0.0/nova/policies/quota_sets.py)
- [base rule](https://github.com/openstack/nova/blob/31.0.0/nova/policies/base.py)
- [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/quota_sets.py)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
- [quota class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/quota_set.py)
- [Quota Set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [Python Quota Query Test](python_quota_query_test.go)
- [Read Test](../../../internal/quotafetch/read_test.go)
