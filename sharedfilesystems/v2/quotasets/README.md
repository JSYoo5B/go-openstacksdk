# Manila project quota sets

This SDK-owned package fills a gap in Gophercloud v2.15.0: that version has no
Manila quota transport package. A quota scope fixes the project once and exposes
the actual singleton operations. Quotas do not have `List`, `Find`, or `Wait`.

| openstacksdk | Go SDK |
| --- | --- |
| `shared_file_system.get_quota_set(project)` | `scope.Get(ctx)` |
| `shared_file_system.update_quota_set(project, shares=0)` | `scope.Update(ctx, quotasets.UpdateOpts{Shares: &zero})` |
| `shared_file_system.revert_quota_set(project)` | `scope.Reset(ctx)` |
| Quota resource fetch using `/defaults` | `scope.Defaults(ctx)` |
| Quota resource fetch using `/detail`, API 2.25+ | `scope.Detail(ctx)` |

Python's pinned `get_quota_set(project, **query)` implementation calls `fetch`
without forwarding `query`. This SDK uses the documented `/detail` operation
for usage, reservations and limits rather than inventing `usage=true`.

```python
quota = conn.shared_file_system.get_quota_set("project-id")
quota = conn.shared_file_system.update_quota_set("project-id", shares=0)
conn.shared_file_system.revert_quota_set("project-id")
```

```go
import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/resource"
    "gophercloudsdk/sharedfilesystems/v2/quotasets"
)

func manageQuota(ctx context.Context, client *gophercloud.ServiceClient) error {
    scope, err := quotasets.New(client).InProject(ctx, resource.ID("project-id"))
    if err != nil {
        return err
    }
    quota, err := scope.Get(ctx)
    if err != nil {
        return err
    }
    _ = quota.Shares // *int64; nil is omitted or null, distinguished by quota.Body
    zero := int64(0)
    _, err = scope.Update(ctx, quotasets.UpdateOpts{Shares: &zero},
        quotasets.WithUpdateForce(false))
    return err
}
```

An ID does not trigger Keystone HTTP. `resource.Name("tenant")` uses the separate
Keystone v3 client supplied through `WithIdentityClient`. Exact names are resolved
across all pages; missing, ambiguous or invalid IDs fail before a Manila request.
`CurrentProject(ctx)` reads the recorded Keystone v2/v3 authentication result.
Manual-token, domain and system authentication need an explicit project ID.

The endpoint's catalog project and the target quota project remain distinct.
The configured service client's resource base is retained; no project is guessed
from its URL. An omitted microversion, or 2.6 and earlier, uses
`os-quota-sets/{project}`; 2.7+ uses `quota-sets/{project}`. `Detail` needs a
configured microversion of at least 2.25. `latest` delegates negotiation to the
server. The SDK never raises the client's microversion automatically.

Limits use exact `int64` values. Nil input is omitted, zero is sent, and -1 means
unlimited; values below -1 fail before HTTP. `WithQuotaOptions` snapshots typed
pointers and nested `Extra` JSON when constructed. `WithUpdateField` snapshots
an extension field without requiring a builder interface. Extensions cannot
replace declared limits, `force`, scope selectors, or response-only metadata.
The server validates extension schemas and microversion-specific quota fields.

`QuotaResource` and `QuotaDetailResource` retain the fixed `ProjectID`, the server's
separate `ID`, response headers, status code and every `quota_set` field as raw
JSON, including nulls and unknown nested fields. Typed usage fields expose
`Limit`, `InUse` and `Reserved` without floating-point integer conversion.

Get/defaults/detail/update require HTTP 200; reset requires HTTP 202. Reset removes
overrides, returns response metadata and does not fetch the result automatically.
The default missing-project policy is strict. `WithResetIgnoreMissing(true)`
suppresses only HTTP 404. Context, HTTP and JSON decoding causes remain available
through `errors.Is`/`errors.As`.

All scoped requests retain their exact method, path, query and origin through
redirects, retries and reauthentication. Configure service clients before sharing
them between concurrent callers.

Contract sources: the pinned openstacksdk
[`QuotaSet`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/quota_set.py),
[`Proxy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/_proxy.py),
and the official [Manila quota API](https://docs.openstack.org/api-ref/shared-file-system/#quota-sets).
