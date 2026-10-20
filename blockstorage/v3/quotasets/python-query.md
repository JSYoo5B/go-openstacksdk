# Python Query

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
quota = conn.block_storage.get_quota_set(
    "project-id", usage=True, fields=["volumes", "gigabytes"]
)
print(quota.volumes, quota.usage, quota.reservation)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    scope, err := conn.BlockStorageProjectQuotas(ctx, resource.ID("project-id"))
    if err != nil { log.Fatal(err) }
    quota, err := scope.GetRecord(ctx,
        quotasets.WithQuotaUsage(true),
        quotasets.WithQuotaReadQuery("fields", []string{"volumes", "gigabytes"}),
    )
    if err != nil { log.Fatal(err) }
    fmt.Println(string(quota.Resource.Body["volumes"]))
    fmt.Println(string(quota.Resource.Body["usage"]), string(quota.Resource.Body["reservation"]))
    fmt.Println(quota.ProjectID, quota.StatusCode)
}
```

## References

- [block storage proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Quotas](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/quotas.py)
- [Base](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/base.py)
- [controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/quotas.py)
- [quota class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/quota_set.py)
- [Quota Set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Python Query Test](python_query_test.go)
- [Read Test](../../../internal/quotafetch/read_test.go)
