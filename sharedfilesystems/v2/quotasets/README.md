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

The Manila API also supports a fixed user or share type within a project. The
pinned Python proxy does not expose these as separate scope helpers; passing
`user_id` or `share_type` to its `get_quota_set` does not forward those queries.
The Go scopes call the documented REST selectors directly.

| Manila REST operation | Go SDK |
| --- | --- |
| `GET/PUT/DELETE /quota-sets/{project}?user_id={user}` | `scope.InUser(ctx, resource.ID(user))`, then `Get`/`Update`/`Reset` |
| `GET /quota-sets/{project}/detail?user_id={user}` | `userScope.Detail(ctx)` |
| `GET/PUT/DELETE /quota-sets/{project}?share_type={type}` | `scope.InShareType(ctx, resource.ID(type))`, then `Get`/`Update`/`Reset` |
| `GET /quota-sets/{project}/detail?share_type={type}` | `typeScope.Detail(ctx)` |

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

`InUser` inherits the project's Keystone client for exact user names and accepts
a `WithIdentityClient` override. `InShareType` resolves exact names using the
Manila service's existing share-type resource collection. Both explicit child
IDs avoid lookup HTTP. Share-type scopes require API 2.39 before a name lookup
or any quota operation; user detail requires API 2.25. The server enforces the
microversions for individual newer quota fields.

```go
import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/resource"
    "gophercloudsdk/sharedfilesystems/v2/quotasets"
)

func manageScopedQuota(ctx context.Context, client *gophercloud.ServiceClient) error {
    parent, err := quotasets.New(client).InProject(ctx, resource.ID("project-id"))
    if err != nil {
        return err
    }
    user, err := parent.InUser(ctx, resource.ID("user-id"))
    if err != nil {
        return err
    }
    count := int64(10)
    if _, err := user.Update(ctx, quotasets.UpdateOpts{Shares: &count}); err != nil {
        return err
    }
    shareType, err := parent.InShareType(ctx, resource.Name("gold")) // API 2.39+
    if err != nil {
        return err
    }
    quota, err := shareType.Get(ctx)
    if err != nil {
        return err
    }
    _ = quota.ShareTypeID // resolved target, separate from quota.ID
    return nil
}
```

Child scopes use composition with a private binding, so they cannot expose a
project-wide reset through embedded methods. They expose `Get`, `Detail`,
`Update` and `Reset`; defaults remain a project operation. Their fixed query
contains exactly one of `user_id` and `share_type`. Share networks are quotas
for projects/users and cannot be updated through a share-type scope.

The endpoint's catalog project and the target quota project remain distinct.
The configured service client's resource base is retained; no project is guessed
from its URL. An omitted microversion, or 2.6 and earlier, uses
`os-quota-sets/{project}`; 2.7+ uses `quota-sets/{project}`. `Detail` needs a
configured numeric microversion of at least 2.25. Symbolic `latest` fails before
lookup or HTTP: it cannot determine whether the server uses legacy or current
routes. Connection range discovery can select a numeric version first. The SDK
never raises the client's microversion automatically.

The selected version is validated against source `MoreHeaders` before lookup
and on every operation. Case-insensitive legacy or general version headers
must agree with the selected version; empty selection means 2.0. Matching source
headers are retained. Native `shared-file-system`, `sharev2` and `share` clients
generate the required legacy Manila header. A manual client with blank `Type`
needs an explicit matching `X-OpenStack-Manila-API-Version`; a general
`OpenStack-API-Version` header alone does not establish the version for Manila.
Conflicts and wrong service types fail before quota or name-lookup HTTP.

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
