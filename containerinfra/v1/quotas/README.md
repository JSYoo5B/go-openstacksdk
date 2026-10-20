# Quotas

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:containerinfra/v1/quotas.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas.ProjectQuotaScope.ForResource`
- `github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas.ResourceQuotaScope.Create`
- `github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas.WithHardLimit`
- `github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas.WithQuotaCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas.WithQuotaCreateField`

- Explicit project IDs and resource names fix the compound quota target; exact project names resolve once through a separate Keystone client, and recorded v2/v3 auth fixes the current project without guessing from endpoints or response identity.
  Tests: `api/magnum_quota_create_test.go:TestMagnumQuotaScopeUsesSeparateIdentityAndRecordedAuth`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`, `connection_project_quotas_test.go:TestConnectionServiceQuotasIDAuthAndCancellation`.
- POST /quotas sends flat project_id/resource/hard_limit JSON with an explicitly supplied hard limit, zero and exact platform integers. Typed pointer and extension options snapshot at construction and remain byte-accurate across reuse and reauth; protected identity/core fields and unsupported options fail before HTTP.
  Tests: `api/magnum_quota_create_test.go:TestMagnumQuotaScopedCreatePreservesFixedIdentityExactJSONAndMetadata`, `api/magnum_quota_create_test.go:TestMagnumQuotaCreateOptionsSnapshotAndRetainIntegersAcrossReauth`, `api/magnum_quota_create_test.go:TestMagnumQuotaCreatePreflightAndCancellation`, `api/magnum_quota_create_test.go:TestMagnumQuotaNilScopesRemainSafeToInspectAndRejectCreate`.
- Only201 succeeds; returned quota preserves native values, exact raw row ID, unknown/null fields, request identity separate from response identity and independent HTTP metadata. Accepted decode failures retain raw bytes/header/status without automatic resend; native HTTP/context causes and fixed-scope redirect boundaries remain observable.
  Tests: `api/magnum_quota_create_test.go:TestMagnumQuotaScopedCreatePreservesFixedIdentityExactJSONAndMetadata`, `api/magnum_quota_create_test.go:TestMagnumQuotaCreateHTTPFailuresAndSuccessPolicy`, `api/magnum_quota_create_test.go:TestMagnumQuotaCreateDecodeFailuresRetainSuccessfulEvidence`, `api/magnum_quota_create_test.go:TestMagnumQuotaCreateRejectsScopeChangingRedirectAndPreservesCanceledCause`, `api/magnum_quota_create_test.go:TestMagnumQuotaGeneratedCreateRetainsNativeContract`.

Python/Go differences:

- Generated native API.Create remains unchanged. The high-level scope owns concrete option snapshots and the fixed project/resource pair, requires an explicit hard limit instead of choosing an administrative default, and returns extension-preserving QuotaResource with HTTP metadata.
- RequestProjectID/RequestResource record the fixed input separately from server ProjectID/Resource. Numeric row IDs are re-read from raw JSON rather than rounded through the native float64 ID representation.
- The pinned Python container-infrastructure proxy has no quota declaration. Additional scoped GET/PATCH/DELETE and paginated List/All REST methods are documented and tested separately, and are not invented native or Python catalog operations.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
func createClusterQuota(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := quotas.New(client)
    project, err := api.InProject(ctx, resource.ID("project-id"))
    if err != nil { return err }
    scope, err := project.ForResource(quotas.Cluster)
    if err != nil { return err }

    quota, err := scope.Create(ctx, quotas.WithHardLimit(0),
        quotas.WithQuotaCreateField("vendor", map[string]string{"tier": "small"}))
    if err != nil { return err }
    fmt.Println(quota.RequestProjectID, quota.RequestResource,
        quota.ID, quota.HardLimit, quota.Header.Get("X-Openstack-Request-Id"))
    current, err := scope.Get(ctx)
    if err != nil { return err }
    fmt.Println(current.HardLimit)

    updated, err := scope.Update(ctx, quotas.WithHardLimit(2))
    if err != nil { return err }
    fmt.Println(updated.StatusCode)

    _, err = scope.Delete(ctx)
    return err
}
```

```go
import (
    "context"
    "fmt"
    "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas"
)

func listQuotas(ctx context.Context, api *quotas.API) error {
    for quota, err := range api.List(ctx,
        quotas.WithListOptions(quotas.ListOpts{Limit: 100}),
        quotas.WithListAllTenants(true),
    ) {
        if err != nil { return err }
        fmt.Println(quota.ID, quota.ProjectID, quota.Resource, quota.HardLimit)
    }
    return nil
}
```

## References

- [pinned native Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/containerinfra/v1/quotas/requests.go)
- [native quota model](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/containerinfra/v1/quotas/results.go)
- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/container_infrastructure_management/v1/_proxy.py)
- [Container Infrastructure Management](https://docs.openstack.org/api-ref/container-infrastructure-management/#magnum-quota-api)
- [server quota controller](https://github.com/openstack/magnum/blob/master/magnum/api/controllers/v1/quota.py)
- [collection next](https://github.com/openstack/magnum/blob/master/magnum/api/controllers/v1/collection.py)
- [server limit validation](https://github.com/openstack/magnum/blob/master/magnum/api/utils.py)
- [Magnum Quota Create Test](../../../api/magnum_quota_create_test.go)
- [Magnum Quota Operations Test](../../../api/magnum_quota_operations_test.go)
- [Magnum Quota List Test](../../../api/magnum_quota_list_test.go)
